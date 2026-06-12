#!/usr/bin/env python3
# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

"""Migrate viper.Set/Get calls in test files to config.Get() direct assignment."""
import sys, os

# 1) graceful_run_test.go - replaces viper.Set + viper.Get with cfg assignment
path1 = 'cmd/server/graceful_run_test.go'
with open(path1, 'r', encoding='utf-8') as f:
    c1 = f.read()

old1 = '''func TestHTTPStartAndGracefulShutdown(t *testing.T) {
\tviper.Set("server.listen.http.addr", "127.0.0.1:0")
\tviper.Set("server.shutdown_timeout", 500*time.Millisecond)

\tshutdownCh := make(chan context.Context)
\tr := BuildEngine(context.Background(), testCfgGrace(), shutdownCh)

\tgr, err := graceful.New(
\t\tr,
\t\tgraceful.WithAddr(viper.GetString("server.listen.http.addr")),
\t\tgraceful.WithShutdownTimeout(viper.GetDuration("server.shutdown_timeout")),
\t)'''

new1 = '''func TestHTTPStartAndGracefulShutdown(t *testing.T) {
\tcfg := config.Get()
\tcfg.Server.Listen.HTTP.Addr = "127.0.0.1:0"
\tcfg.Server.ShutdownTimeout = "500ms"

\tshutdownCh := make(chan context.Context)
\tr := BuildEngine(context.Background(), testCfgGrace(), shutdownCh)

\tgr, err := graceful.New(
\t\tr,
\t\tgraceful.WithAddr(cfg.Server.Listen.HTTP.Addr),
\t\tgraceful.WithShutdownTimeout(500*time.Millisecond),
\t)'''

if old1 in c1:
    c1 = c1.replace(old1, new1)
    print('1) graceful_run_test.go migrated')
else:
    print('1) ERROR: graceful pattern not found')

# Also remove unnecessary viper import
old_viper_import = '''\t"github.com/cocomhub/cocom/internal/config"
\t"github.com/gin-contrib/graceful"
\t"github.com/spf13/viper"'''

new_viper_import = '''\t"github.com/cocomhub/cocom/internal/config"
\t"github.com/gin-contrib/graceful"'''

if old_viper_import in c1:
    c1 = c1.replace(old_viper_import, new_viper_import)
    print('   Import cleaned')

with open(path1, 'w', encoding='utf-8') as f:
    f.write(c1)

# 2) middleware_test.go - use cfg directly
path2 = 'cmd/server/middleware_test.go'
with open(path2, 'r', encoding='utf-8') as f:
    c2 = f.read()

# Replace viper.Set calls with config assignment
old2 = '''func TestCORSAndGzip(t *testing.T) {
\tviper.Set("server.cors.enabled", true)
\tviper.Set("server.cors.allow_origins", "*")
\tviper.Set("server.cors.allow_methods", "GET,POST,DELETE,OPTIONS")
\tviper.Set("server.cors.allow_headers", "X-Requested-With,Content-Type")
\tviper.Set("server.gzip.enabled", true)'''

new2 = '''func TestCORSAndGzip(t *testing.T) {
\tcfg := config.Get()
\tcfg.Server.CORS = config.CORSCfg{Enabled: true, AllowOrigins: "*", AllowMethods: "GET,POST,DELETE,OPTIONS", AllowHeaders: "X-Requested-With,Content-Type"}
\tcfg.Server.Gzip = config.GzipCfg{Enabled: true, Level: 1}'''

if old2 in c2:
    c2 = c2.replace(old2, new2)
    print('2) middleware_test.go TestCORSAndGzip migrated')
else:
    print('2) ERROR: TestCORSAndGzip pattern not found')

old2b = '''func TestMaxBodySize(t *testing.T) {
\tviper.Set("server.cors.enabled", false)
\tviper.Set("server.gzip.enabled", false)'''

new2b = '''func TestMaxBodySize(t *testing.T) {
\tcfg := config.Get()
\tcfg.Server.CORS = config.CORSCfg{}
\tcfg.Server.Gzip = config.GzipCfg{}'''

if old2b in c2:
    c2 = c2.replace(old2b, new2b)
    print('   TestMaxBodySize migrated')
else:
    print('   TestMaxBodySize pattern not found')

old2c = '''func TestRateLimit(t *testing.T) {
\tviper.Set("server.ratelimit.enabled", true)
\tviper.Set("server.ratelimit.rps", 1)
\tviper.Set("server.ratelimit.burst", 1)'''

new2c = '''func TestRateLimit(t *testing.T) {
\tcfg := config.Get()
\tcfg.Server.RateLimit = config.RateLimitCfg{Enabled: true, RPS: 1, Burst: 1}'''

if old2c in c2:
    c2 = c2.replace(old2c, new2c)
    print('   TestRateLimit migrated')
else:
    print('   TestRateLimit pattern not found')

old2d = '''\tviper.Set("server.ratelimit.enabled", false)'''

if old2d in c2:
    c2 = c2.replace(old2d, '\tcfg.Server.RateLimit = config.RateLimitCfg{}')
    print('   RateLimit teardown migrated')

# Fix import
old_imp2 = '''\t"github.com/cocomhub/cocom/cmd/server/internal/testutil"
\t"github.com/spf13/viper"'''

new_imp2 = '''\t"github.com/cocomhub/cocom/cmd/server/internal/testutil"
\t"github.com/cocomhub/cocom/internal/config"'''

if old_imp2 in c2:
    c2 = c2.replace(old_imp2, new_imp2)
    print('   middleware_test import cleaned')

with open(path2, 'w', encoding='utf-8') as f:
    f.write(c2)

# 3) pprof_test.go
path3 = 'cmd/server/pprof_test.go'
with open(path3, 'r', encoding='utf-8') as f:
    c3 = f.read()

old3 = '''\tviper.Set("debug.allow_remote", false)
\tr := BuildEngine(context.Background(), testutil.TestServerConfig(), nil)'''

# This is trickier - pprof uses a different key "debug.allow_remote" not "admin.allow_remote"
# The middlewares.LocalGuard accepts generic viper key, still reads from viper
# So we need to keep viper.Set for this one since LocalGuard reads from viper directly
# Unless we add debug.allow_remote to the config struct too
# For now, skip this one and add it as a note

print('3) pprof_test.go uses debug.allow_remote - kept viper.Set for now (different config key)')

# 4) settings_integration_test.go
path4 = 'cmd/server/settings_integration_test.go'
with open(path4, 'r', encoding='utf-8') as f:
    c4 = f.read()

# Already partially edited - find current viper.Set line
old4 = '''\tviper.Set("server.ratelimit.enabled", false)
\tr := BuildEngine(context.Background(), testutil.TestServerConfigMinimal(), nil)'''

new4 = '''\tcfg := config.Get()
\tcfg.Server.RateLimit = config.RateLimitCfg{}
\tr := BuildEngine(context.Background(), testutil.TestServerConfigMinimal(), nil)'''

if old4 in c4:
    c4 = c4.replace(old4, new4)
    print('4) settings_integration_test.go migrated')
else:
    # Check current state
    idx = c4.find('TestSettingsV1AndAlias')
    if idx >= 0:
        print('   Current TestSettingsV1AndAlias context:')
        print(c4[idx:idx+200])
    else:
        print('4) ERROR: TestSettingsV1AndAlias not found')

with open(path4, 'w', encoding='utf-8') as f:
    f.write(c4)

print('\n--- Done ---')
