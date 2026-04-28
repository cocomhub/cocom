// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoIndexStore struct {
	coll            *mongo.Collection
	idField         string
	nameField       string
	prefix          string
	filterBuilder   func(IndexFilter) bson.M
	encode          func(*ArchiveMeta) (any, error)
	decode          func(any) (*ArchiveMeta, error)
	embedded        bool
	requireExisting bool
}

type MongoOption func(*mongoIndexStore)

func WithMongoPrefix(prefix string) MongoOption {
	return func(m *mongoIndexStore) { m.prefix = prefix }
}

func WithMongoIDField(field string) MongoOption {
	return func(m *mongoIndexStore) { m.idField = field }
}

func WithMongoNameField(field string) MongoOption {
	return func(m *mongoIndexStore) { m.nameField = field }
}

func WithMongoEncoder(enc func(*ArchiveMeta) (any, error)) MongoOption {
	return func(m *mongoIndexStore) { m.encode = enc }
}

func WithMongoDecoder(dec func(any) (*ArchiveMeta, error)) MongoOption {
	return func(m *mongoIndexStore) { m.decode = dec }
}

func WithMongoRequireExisting() MongoOption {
	return func(m *mongoIndexStore) { m.requireExisting = true }
}

func NewMongoIndexStore(coll *mongo.Collection, opts ...MongoOption) IndexStore {
	m := &mongoIndexStore{
		coll:      coll,
		idField:   "id",
		nameField: "name",
	}
	m.filterBuilder = m.defaultFilter
	m.encode = m.defaultEncode
	m.decode = m.defaultDecode
	for _, o := range opts {
		o(m)
	}
	if m.prefix != "" {
		m.embedded = true
	}
	return m
}

func NewComicInfoArchiveIndexStore(coll *mongo.Collection) IndexStore {
	m := NewMongoIndexStore(
		coll,
		WithMongoIDField("cid"),
		WithMongoPrefix("archive"),
		WithMongoRequireExisting(),
	).(*mongoIndexStore)
	m.encode = m.encodeComicInfoArchive
	m.decode = m.decodeComicInfoArchive
	return m
}

func (m *mongoIndexStore) keyPath(field string) string {
	if m.prefix == "" {
		return field
	}
	return m.prefix + "." + field
}

func (m *mongoIndexStore) defaultFilter(f IndexFilter) bson.M {
	q := bson.M{}
	if f.ID != 0 {
		q[m.idField] = f.ID
	}
	if f.Name != "" {
		q[m.keyPath(m.nameField)] = f.Name
	}
	if !f.Before.IsZero() || !f.After.IsZero() {
		r := bson.M{}
		if !f.After.IsZero() {
			r["$gt"] = f.After
		}
		if !f.Before.IsZero() {
			r["$lt"] = f.Before
		}
		if len(r) > 0 {
			q[m.keyPath("mod_time")] = r
		}
	}
	return q
}

func (m *mongoIndexStore) defaultEncode(meta *ArchiveMeta) (any, error) {
	doc, err := util.ToMap(meta)
	if err != nil {
		return nil, err
	}
	if m.embedded {
		return bson.M{
			m.idField: meta.ID,
			m.prefix:  doc,
		}, nil
	}
	doc[m.idField] = meta.ID
	return bson.M(doc), nil
}

func ArchiveMeta2CocomArchiveInfo(meta *ArchiveMeta) (*api.ArchiveInfo, error) {
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	archiveInfo := &api.ArchiveInfo{
		Path:          meta.Path,
		Size:          meta.Size,
		CreatedAt:     meta.ModTime,
		Algorithm:     string(meta.Type),
		Locators:      meta.Locators,
		ReplicaHealth: meta.ReplicaHealth,
	}
	if meta.Checksum.Algorithm == "md5" && meta.Checksum.Value != "" {
		archiveInfo.MD5 = meta.Checksum.Value
	} else {
		archiveInfo.MD5 = ""
	}
	return archiveInfo, nil
}

func (m *mongoIndexStore) encodeComicInfoArchive(meta *ArchiveMeta) (any, error) {
	archiveInfo, err := ArchiveMeta2CocomArchiveInfo(meta)
	if err != nil {
		return nil, err
	}
	return bson.M{
		m.idField: meta.ID,
		m.prefix:  archiveInfo,
	}, nil
}

func (m *mongoIndexStore) defaultDecode(v any) (*ArchiveMeta, error) {
	switch t := v.(type) {
	case bson.M:
		if m.embedded {
			sub, _ := t[m.prefix].(bson.M)
			return m.decodeFromMap(sub)
		}
		return m.decodeFromMap(t)
	case map[string]any:
		if m.embedded {
			sub, _ := t[m.prefix].(map[string]any)
			return m.decodeFromMap(bson.M(sub))
		}
		return m.decodeFromMap(bson.M(t))
	default:
		return nil, fmt.Errorf("%w: decode invalid type: %T", ErrInternal, v)
	}
}

var regexComicInfoArchivePath = regexp.MustCompile(`^.*/(\d+)\.cocoma$`)

func (m *mongoIndexStore) decodeComicInfoArchive(v any) (*ArchiveMeta, error) {
	doc, ok := asBSONMap(v)
	if !ok {
		return nil, fmt.Errorf("%w: decode comic info archive invalid type: %T", ErrInternal, v)
	}
	archiveDoc, ok := mapFromMap(doc, m.prefix)
	if !ok || archiveDoc == nil {
		return nil, fmt.Errorf("%w: decode comic info archive not found: prefix=%s", ErrNotFound, m.prefix)
	}

	archiveInfo, err := m.decodeComicInfoArchiveFromMap(archiveDoc)
	if err != nil {
		return nil, err
	}
	if archiveInfo == nil {
		return nil, fmt.Errorf("%w: decode comic info archive from map not found", ErrNotFound)
	}
	match := regexComicInfoArchivePath.FindStringSubmatch(archiveInfo.Path)
	if match == nil || len(match) != 2 {
		return nil, fmt.Errorf("%w: parse comic info archive path failed", ErrNotFound)
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, fmt.Errorf("%w: convert comic info archive id failed", ErrNotFound)
	}
	return &ArchiveMeta{
		ID:            id,
		Name:          archiveInfo.Path + ".origin",
		Path:          archiveInfo.Path,
		Size:          archiveInfo.Size,
		FileCount:     -1,
		ModTime:       archiveInfo.CreatedAt,
		Version:       1,
		Type:          archiveTypeFromString(archiveInfo.Algorithm),
		Checksum:      storage.Checksum{Algorithm: "md5", Value: archiveInfo.MD5},
		Locators:      archiveInfo.Locators,
		ReplicaHealth: archiveInfo.ReplicaHealth,
	}, nil
}

func (m *mongoIndexStore) decodeFromMap(mp bson.M) (*ArchiveMeta, error) {
	if mp == nil {
		return nil, ErrNotFound
	}
	var meta ArchiveMeta
	if v, ok := mp["id"].(int32); ok {
		meta.ID = int(v)
	} else if v, ok := mp["id"].(int64); ok {
		meta.ID = int(v)
	} else if v, ok := mp["id"].(int); ok {
		meta.ID = v
	} else if v, ok := mp["id"].(float64); ok {
		meta.ID = int(v)
	}
	if v, ok := mp["name"].(string); ok {
		meta.Name = v
	}
	if v, ok := mp["path"].(string); ok {
		meta.Path = v
	}
	if v, ok := mp["size"].(int64); ok {
		meta.Size = v
	} else if v, ok := mp["size"].(int32); ok {
		meta.Size = int64(v)
	} else if v, ok := mp["size"].(float64); ok {
		meta.Size = int64(v)
	}
	if v, ok := mp["file_count"].(int32); ok {
		meta.FileCount = int(v)
	} else if v, ok := mp["file_count"].(int64); ok {
		meta.FileCount = int(v)
	} else if v, ok := mp["file_count"].(int); ok {
		meta.FileCount = v
	} else if v, ok := mp["file_count"].(float64); ok {
		meta.FileCount = int(v)
	}
	if v, ok := timeFromMap(mp, "mod_time"); ok {
		meta.ModTime = v
	}
	if v, ok := mp["version"].(int32); ok {
		meta.Version = int(v)
	} else if v, ok := mp["version"].(int64); ok {
		meta.Version = int(v)
	} else if v, ok := mp["version"].(int); ok {
		meta.Version = v
	} else if v, ok := mp["version"].(float64); ok {
		meta.Version = int(v)
	}
	if v, ok := mp["type"].(string); ok {
		meta.Type = archiveTypeFromString(v)
	}
	if v, ok := mp["checksum"]; ok {
		if checksum, ok := decodeChecksum(v); ok {
			meta.Checksum = checksum
		}
	}
	if v, ok := mp["locators"]; ok {
		if locators, ok := decodeLocators(v); ok {
			meta.Locators = locators
		}
	}
	if health, ok := decodeReplicaHealth(mp); ok {
		meta.ReplicaHealth = health
	}
	return &meta, nil
}

func (m *mongoIndexStore) decodeComicInfoArchiveFromMap(mp bson.M) (*api.ArchiveInfo, error) {
	archiveInfo := &api.ArchiveInfo{}
	if mp == nil {
		return archiveInfo, ErrNotFound
	}
	if v, ok := stringFromMap(mp, "path"); ok {
		archiveInfo.Path = v
	}
	if v, ok := stringFromMap(mp, "md5"); ok {
		archiveInfo.MD5 = v
	}
	if v, ok := int64FromMap(mp, "size"); ok {
		archiveInfo.Size = v
	}
	if v, ok := timeFromMap(mp, "created_at", "mod_time"); ok {
		archiveInfo.CreatedAt = v
	}
	if v, ok := stringFromMap(mp, "algorithm", "type"); ok {
		archiveInfo.Algorithm = v
	}
	if v, ok := boolFromMap(mp, "by_force"); ok {
		archiveInfo.ByForce = v
	}
	if v, ok := mp["locators"]; ok {
		if locators, ok := decodeLocators(v); ok {
			archiveInfo.Locators = locators
		}
	}
	if health, ok := decodeReplicaHealth(mp); ok {
		archiveInfo.ReplicaHealth = health
	}
	return archiveInfo, nil
}

func decodeChecksum(v any) (storage.Checksum, bool) {
	switch t := v.(type) {
	case storage.Checksum:
		return t, true
	case *storage.Checksum:
		if t == nil {
			return storage.Checksum{}, false
		}
		return *t, true
	case bson.M:
		return checksumFromMap(t)
	case map[string]any:
		return checksumFromMap(bson.M(t))
	default:
		return storage.Checksum{}, false
	}
}

func checksumFromMap(mp bson.M) (storage.Checksum, bool) {
	var checksum storage.Checksum
	if mp == nil {
		return checksum, false
	}
	if v, ok := stringFromMap(mp, "algorithm", "alg"); ok {
		checksum.Algorithm = v
	}
	if v, ok := stringFromMap(mp, "value"); ok {
		checksum.Value = v
	}
	return checksum, checksum != (storage.Checksum{})
}

func decodeLocators(v any) ([]storage.StorageLocator, bool) {
	switch t := v.(type) {
	case []storage.StorageLocator:
		return append([]storage.StorageLocator(nil), t...), true
	case primitive.A:
		res := make([]storage.StorageLocator, 0, len(t))
		for _, item := range t {
			loc, ok := decodeLocator(item)
			if !ok {
				continue
			}
			res = append(res, loc)
		}
		return res, true
	case []any:
		res := make([]storage.StorageLocator, 0, len(t))
		for _, item := range t {
			loc, ok := decodeLocator(item)
			if !ok {
				continue
			}
			res = append(res, loc)
		}
		return res, true
	default:
		return nil, false
	}
}

func decodeLocator(v any) (storage.StorageLocator, bool) {
	switch t := v.(type) {
	case storage.StorageLocator:
		return t, true
	case *storage.StorageLocator:
		if t == nil {
			return storage.StorageLocator{}, false
		}
		return *t, true
	case bson.M:
		return locatorFromMap(t)
	case map[string]any:
		return locatorFromMap(bson.M(t))
	default:
		return storage.StorageLocator{}, false
	}
}

func locatorFromMap(mp bson.M) (storage.StorageLocator, bool) {
	var loc storage.StorageLocator
	if mp == nil {
		return loc, false
	}
	if v, ok := stringFromMap(mp, "backend", "store"); ok {
		loc.Backend = v
	}
	if v, ok := stringFromMap(mp, "key"); ok {
		loc.Key = v
	}
	if health, ok := decodeReplicaHealth(mp); ok {
		loc.ReplicaHealth = health
	} else if nested, ok := mapFromMap(mp, "replicaHealth", "replicahealth"); ok {
		if health, ok := decodeReplicaHealth(nested); ok {
			loc.ReplicaHealth = health
		}
	}
	return loc, loc != (storage.StorageLocator{})
}

func decodeReplicaHealth(v any) (storage.ReplicaHealth, bool) {
	switch t := v.(type) {
	case storage.ReplicaHealth:
		return t, true
	case *storage.ReplicaHealth:
		if t == nil {
			return storage.ReplicaHealth{}, false
		}
		return *t, true
	case bson.M:
		return replicaHealthFromMap(t)
	case map[string]any:
		return replicaHealthFromMap(bson.M(t))
	default:
		return storage.ReplicaHealth{}, false
	}
}

func asBSONMap(v any) (bson.M, bool) {
	switch t := v.(type) {
	case bson.M:
		return t, true
	case map[string]any:
		return bson.M(t), true
	default:
		return nil, false
	}
}

func replicaHealthFromMap(mp bson.M) (storage.ReplicaHealth, bool) {
	var health storage.ReplicaHealth
	if mp == nil {
		return health, false
	}
	if v, ok := boolFromMap(mp, "healthy"); ok {
		health.Healthy = v
	}
	if v, ok := timeFromMap(mp, "checked_at"); ok {
		health.CheckedAt = v
	}
	return health, health != (storage.ReplicaHealth{})
}

func stringFromMap(mp bson.M, keys ...string) (string, bool) {
	for _, key := range keys {
		if v, ok := mp[key].(string); ok {
			return v, true
		}
	}
	return "", false
}

func intFromMap(mp bson.M, keys ...string) (int, bool) {
	for _, key := range keys {
		switch v := mp[key].(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		}
	}
	return 0, false
}

func int64FromMap(mp bson.M, keys ...string) (int64, bool) {
	for _, key := range keys {
		switch v := mp[key].(type) {
		case int:
			return int64(v), true
		case int32:
			return int64(v), true
		case int64:
			return v, true
		case float64:
			return int64(v), true
		}
	}
	return 0, false
}

func boolFromMap(mp bson.M, keys ...string) (bool, bool) {
	for _, key := range keys {
		if v, ok := mp[key].(bool); ok {
			return v, true
		}
	}
	return false, false
}

func timeFromMap(mp bson.M, keys ...string) (time.Time, bool) {
	for _, key := range keys {
		switch v := mp[key].(type) {
		case time.Time:
			return v, true
		case primitive.DateTime:
			return v.Time(), true
		case string:
			t, err := time.Parse(time.RFC3339Nano, v)
			if err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func mapFromMap(mp bson.M, keys ...string) (bson.M, bool) {
	for _, key := range keys {
		switch v := mp[key].(type) {
		case bson.M:
			return v, true
		case map[string]any:
			return bson.M(v), true
		}
	}
	return nil, false
}

func flattenBSON(prefix string, doc bson.M, setDoc, unsetDoc bson.M) {
	for key, raw := range doc {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := raw.(type) {
		case nil:
			unsetDoc[fullKey] = ""
		case bson.M:
			flattenBSON(fullKey, v, setDoc, unsetDoc)
		case map[string]any:
			flattenBSON(fullKey, bson.M(v), setDoc, unsetDoc)
		default:
			setDoc[fullKey] = raw
		}
	}
}

func (m *mongoIndexStore) embeddedUpdateDocuments(meta *ArchiveMeta) (bson.M, bson.M, error) {
	payload, err := m.encode(meta)
	if err != nil {
		return nil, nil, err
	}
	root, ok := payload.(bson.M)
	if !ok {
		return nil, nil, fmt.Errorf("%w: encode archive meta failed", ErrInternal)
	}
	var archiveDoc bson.M
	switch v := root[m.prefix].(type) {
	case bson.M:
		archiveDoc = v
	case map[string]any:
		archiveDoc = bson.M(v)
	default:
		obj, err := util.ToMap(v)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s", ErrInternal, err)
		}
		archiveDoc = bson.M(obj)
	}
	setDoc := bson.M{}
	unsetDoc := bson.M{}
	flattenBSON(m.prefix, archiveDoc, setDoc, unsetDoc)
	return setDoc, unsetDoc, nil
}

func (m *mongoIndexStore) Create(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	if m.embedded {
		filter := bson.M{m.idField: meta.ID}
		proj := options.FindOne().SetProjection(bson.M{m.prefix: 1})
		res := m.coll.FindOne(ctx, filter, proj)
		var dst bson.M
		if err := res.Decode(&dst); err != nil {
			if err == mongo.ErrNoDocuments && m.requireExisting {
				return fmt.Errorf("mongo: create err %w: %s=%d", ErrNotFound, m.idField, meta.ID)
			}
			if err == mongo.ErrNoDocuments {
				docAny, encErr := m.encode(meta)
				if encErr != nil {
					return encErr
				}
				_, encErr = m.coll.InsertOne(ctx, docAny)
				return encErr
			}
			return err
		}
		if sub, ok := dst[m.prefix]; ok && sub != nil {
			return fmt.Errorf("mongo: create err %w: %s=%d %s=%+v", ErrAlreadyExists, m.idField, meta.ID, m.prefix, sub)
		}
		setDoc, unsetDoc, err := m.embeddedUpdateDocuments(meta)
		if err != nil {
			return err
		}
		update := bson.M{}
		if len(setDoc) > 0 {
			update["$set"] = setDoc
		}
		if len(unsetDoc) > 0 {
			update["$unset"] = unsetDoc
		}
		_, err = m.coll.UpdateOne(ctx, filter, update)
		return err
	}
	count, err := m.coll.CountDocuments(ctx, bson.M{m.idField: meta.ID})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrAlreadyExists
	}
	docAny, err := m.encode(meta)
	if err != nil {
		return err
	}
	_, err = m.coll.InsertOne(ctx, docAny)
	return err
}

func (m *mongoIndexStore) Get(ctx context.Context, id int) (*ArchiveMeta, error) {
	filter := bson.M{m.idField: id}
	if m.embedded {
		opt := options.FindOne().SetProjection(bson.M{m.prefix: 1})
		var doc bson.M
		err := m.coll.FindOne(ctx, filter, opt).Decode(&doc)
		if err != nil {
			return nil, fmt.Errorf("mongo: get err %w: %s=%d, %s", ErrNotFound, m.idField, id, err.Error())
		}
		return m.decode(doc)
	}
	var doc bson.M
	err := m.coll.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("mongo: decode err: %s=%d, %w", m.idField, id, err)
	}
	return m.decode(doc)
}

func (m *mongoIndexStore) Update(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	if m.embedded {
		setDoc, unsetDoc, err := m.embeddedUpdateDocuments(meta)
		if err != nil {
			return err
		}
		upd := bson.M{}
		if len(setDoc) > 0 {
			upd["$set"] = setDoc
		}
		if len(unsetDoc) > 0 {
			upd["$unset"] = unsetDoc
		}
		res, err := m.coll.UpdateOne(ctx, bson.M{m.idField: meta.ID}, upd)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return ErrNotFound
		}
		return nil
	}
	payload, err := m.encode(meta)
	if err != nil {
		return err
	}
	res, err := m.coll.ReplaceOne(ctx, bson.M{m.idField: meta.ID}, payload)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *mongoIndexStore) Delete(ctx context.Context, id int) error {
	if m.embedded {
		res, err := m.coll.UpdateOne(ctx, bson.M{m.idField: id}, bson.M{"$unset": bson.M{m.prefix: ""}})
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 && m.requireExisting {
			return ErrNotFound
		}
		return err
	}
	_, err := m.coll.DeleteOne(ctx, bson.M{m.idField: id})
	return err
}

func (m *mongoIndexStore) List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error) {
	filter := m.filterBuilder(f)
	opts := options.Find().SetSort(bson.D{{Key: m.idField, Value: 1}})
	cur, err := m.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var res []ArchiveMeta
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		mm, err := m.decode(doc)
		if err != nil {
			continue
		}
		if f.ID != 0 && mm.ID != f.ID {
			continue
		}
		if f.Name != "" && mm.Name != f.Name {
			continue
		}
		if !f.Before.IsZero() && !mm.ModTime.Before(f.Before) {
			continue
		}
		if !f.After.IsZero() && !mm.ModTime.After(f.After) {
			continue
		}
		res = append(res, *mm)
	}
	_ = cur.Close(ctx)
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}

func must(v any, err error) any {
	if err != nil {
		panic(err)
	}
	return v
}

func archiveTypeFromString(s string) (t archive.Type) {
	switch s {
	case string(archive.TypeSingle):
		return archive.TypeSingle
	case string(archive.TypeDouble):
		return archive.TypeDouble
	default:
		return archive.TypeDouble
	}
}
