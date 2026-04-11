// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"sort"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoIndexStore struct {
	coll            *mongo.Collection
	idField         string
	nameField       string
	modTimeField    string
	prefix          string
	filterBuilder   func(IndexFilter) bson.M
	encode          func(ArchiveMeta) (any, error)
	decode          func(any) (ArchiveMeta, error)
	embedded        bool
	requireExisting bool
}

type MongoOption func(*mongoIndexStore)

func WithPrefix(prefix string) MongoOption {
	return func(m *mongoIndexStore) { m.prefix = prefix }
}

func WithIDField(field string) MongoOption {
	return func(m *mongoIndexStore) { m.idField = field }
}

func WithNameField(field string) MongoOption {
	return func(m *mongoIndexStore) { m.nameField = field }
}

func WithModTimeField(field string) MongoOption {
	return func(m *mongoIndexStore) { m.modTimeField = field }
}

func WithFilterBuilder(b func(IndexFilter) bson.M) MongoOption {
	return func(m *mongoIndexStore) { m.filterBuilder = b }
}

func WithEncoder(enc func(ArchiveMeta) (any, error)) MongoOption {
	return func(m *mongoIndexStore) { m.encode = enc }
}

func WithDecoder(dec func(any) (ArchiveMeta, error)) MongoOption {
	return func(m *mongoIndexStore) { m.decode = dec }
}

func NewMongoIndexStore(coll *mongo.Collection, opts ...MongoOption) IndexStore {
	m := &mongoIndexStore{
		coll:         coll,
		idField:      "id",
		nameField:    "name",
		modTimeField: "modTime",
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
		WithIDField("cid"),
		WithPrefix("archive"),
	).(*mongoIndexStore)
	m.requireExisting = true
	m.filterBuilder = func(f IndexFilter) bson.M {
		res := bson.M{}
		if f.ID != 0 {
			res["cid"] = f.ID
		}
		if f.Name != "" {
			res["archive.manager.name"] = f.Name
		}
		if !f.Before.IsZero() || !f.After.IsZero() {
			rangeQ := bson.M{}
			if !f.After.IsZero() {
				rangeQ["$gt"] = f.After
			}
			if !f.Before.IsZero() {
				rangeQ["$lt"] = f.Before
			}
			if len(rangeQ) > 0 {
				res["archive.manager.modTime"] = rangeQ
			}
		}
		return res
	}
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
			q[m.keyPath(m.modTimeField)] = r
		}
	}
	return q
}

func (m *mongoIndexStore) archiveMetaDocument(meta ArchiveMeta) bson.M {
	return bson.M{
		"id":        meta.ID,
		"name":      meta.Name,
		"path":      meta.Path,
		"size":      meta.Size,
		"fileCount": meta.FileCount,
		"modTime":   meta.ModTime,
		"version":   meta.Version,
		"type":      meta.Type,
		"checksum":  meta.Checksum,
		"locators":  meta.Locators,
		"health":    meta.Health,
	}
}

func (m *mongoIndexStore) defaultEncode(meta ArchiveMeta) (any, error) {
	doc := m.archiveMetaDocument(meta)
	if m.embedded {
		return bson.M{
			m.idField: meta.ID,
			m.prefix:  doc,
		}, nil
	}
	doc[m.idField] = meta.ID
	return doc, nil
}

func (m *mongoIndexStore) encodeComicInfoArchive(meta ArchiveMeta) (any, error) {
	archiveDoc := bson.M{
		"path":      meta.Path,
		"size":      meta.Size,
		"algorithm": string(meta.Type),
		"manager":   m.archiveMetaDocument(meta),
	}
	if meta.Checksum.Algorithm == "md5" && meta.Checksum.Value != "" {
		archiveDoc["md5"] = meta.Checksum.Value
	} else {
		archiveDoc["md5"] = nil
	}
	return bson.M{
		m.idField: meta.ID,
		m.prefix:  archiveDoc,
	}, nil
}

func (m *mongoIndexStore) defaultDecode(v any) (ArchiveMeta, error) {
	now := time.Time{}
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
		_ = now
	}
	return ArchiveMeta{}, ErrInternal
}

func (m *mongoIndexStore) decodeComicInfoArchive(v any) (ArchiveMeta, error) {
	doc, ok := asBSONMap(v)
	if !ok {
		return ArchiveMeta{}, ErrInternal
	}
	archiveDoc, ok := mapFromMap(doc, m.prefix)
	if !ok || archiveDoc == nil {
		return ArchiveMeta{}, ErrNotFound
	}

	var meta ArchiveMeta
	if managerDoc, ok := mapFromMap(archiveDoc, "manager", "index"); ok {
		decoded, err := m.decodeFromMap(managerDoc)
		if err == nil {
			meta = decoded
		}
	}

	if meta.ID == 0 {
		if id, ok := intFromMap(doc, m.idField); ok {
			meta.ID = id
		}
	}
	if meta.Path == "" {
		if v, ok := stringFromMap(archiveDoc, "path"); ok {
			meta.Path = v
		}
	}
	if meta.Size == 0 {
		if v, ok := int64FromMap(archiveDoc, "size"); ok {
			meta.Size = v
		}
	}
	if meta.ModTime.IsZero() {
		if v, ok := timeFromMap(archiveDoc, "created_at", "modTime"); ok {
			meta.ModTime = v
		}
	}
	if meta.Type == "" {
		if v, ok := stringFromMap(archiveDoc, "algorithm", "type"); ok {
			meta.Type = archiveTypeFromString(v)
		}
	}
	if meta.Checksum == (storage.Checksum{}) {
		if v, ok := stringFromMap(archiveDoc, "md5"); ok && v != "" {
			meta.Checksum = storage.Checksum{Algorithm: "md5", Value: v}
		}
	}
	return meta, nil
}

func (m *mongoIndexStore) decodeFromMap(mp bson.M) (ArchiveMeta, error) {
	var meta ArchiveMeta
	if mp == nil {
		return ArchiveMeta{}, ErrNotFound
	}
	if v, ok := mp["id"].(int32); ok {
		meta.ID = int(v)
	} else if v, ok := mp["id"].(int64); ok {
		meta.ID = int(v)
	} else if v, ok := mp["id"].(int); ok {
		meta.ID = v
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
	}
	if v, ok := mp["fileCount"].(int32); ok {
		meta.FileCount = int(v)
	} else if v, ok := mp["fileCount"].(int64); ok {
		meta.FileCount = int(v)
	} else if v, ok := mp["fileCount"].(int); ok {
		meta.FileCount = v
	}
	if v, ok := mp["modTime"].(time.Time); ok {
		meta.ModTime = v
	}
	if v, ok := mp["version"].(int32); ok {
		meta.Version = int(v)
	} else if v, ok := mp["version"].(int64); ok {
		meta.Version = int(v)
	} else if v, ok := mp["version"].(int); ok {
		meta.Version = v
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
	if v, ok := mp["health"]; ok {
		if health, ok := decodeReplicaHealth(v); ok {
			meta.Health = health
		}
	}
	return meta, nil
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
	if v, ok := timeFromMap(mp, "checkedAt", "checkedat"); ok {
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
		if v, ok := mp[key].(time.Time); ok {
			return v, true
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

func (m *mongoIndexStore) embeddedUpdateDocuments(meta ArchiveMeta) (bson.M, bson.M, error) {
	payload, err := m.encode(meta)
	if err != nil {
		return nil, nil, err
	}
	root, ok := payload.(bson.M)
	if !ok {
		return nil, nil, ErrInternal
	}
	archiveDoc, ok := root[m.prefix].(bson.M)
	if !ok {
		if archiveMap, ok2 := root[m.prefix].(map[string]any); ok2 {
			archiveDoc = bson.M(archiveMap)
		} else {
			return nil, nil, ErrInternal
		}
	}
	setDoc := bson.M{}
	unsetDoc := bson.M{}
	flattenBSON(m.prefix, archiveDoc, setDoc, unsetDoc)
	return setDoc, unsetDoc, nil
}

func (m *mongoIndexStore) Create(ctx context.Context, meta ArchiveMeta) error {
	if meta.ID == 0 {
		return ErrInvalidArgument
	}
	if m.embedded {
		filter := bson.M{m.idField: meta.ID}
		proj := options.FindOne().SetProjection(bson.M{m.prefix: 1})
		res := m.coll.FindOne(ctx, filter, proj)
		var dst bson.M
		if err := res.Decode(&dst); err != nil {
			if err == mongo.ErrNoDocuments && m.requireExisting {
				return ErrNotFound
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
			return ErrAlreadyExists
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

func (m *mongoIndexStore) Get(ctx context.Context, id int) (ArchiveMeta, error) {
	filter := bson.M{m.idField: id}
	if m.embedded {
		opt := options.FindOne().SetProjection(bson.M{m.prefix: 1})
		var doc bson.M
		err := m.coll.FindOne(ctx, filter, opt).Decode(&doc)
		if err != nil {
			return ArchiveMeta{}, ErrNotFound
		}
		return m.decode(doc)
	}
	var doc bson.M
	err := m.coll.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return ArchiveMeta{}, ErrNotFound
	}
	return m.decode(doc)
}

func (m *mongoIndexStore) Update(ctx context.Context, meta ArchiveMeta) error {
	if meta.ID == 0 {
		return ErrInvalidArgument
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
		res = append(res, mm)
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
