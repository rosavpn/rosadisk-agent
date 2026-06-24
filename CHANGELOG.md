# Changelog

## [0.15.1](https://github.com/rosavpn/rosadisk-agent/compare/v0.15.0...v0.15.1) (2026-06-24)


### Bug Fixes

* handle eventBus.Shutdown error in Worker.Shutdown ([e1dd4a6](https://github.com/rosavpn/rosadisk-agent/commit/e1dd4a6d55ffb6f95bf3656864573d300d4d1285))

## [0.15.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.14.0...v0.15.0) (2026-06-16)


### Features

* add scrub/balance implementation and job logs endpoints ([200a514](https://github.com/rosavpn/rosadisk-agent/commit/200a51417059254ba8ba7f4ab8c8880964b59598))
* add scrub/balance implementation and job logs endpoints ([8f35ca3](https://github.com/rosavpn/rosadisk-agent/commit/8f35ca30065ebd8d90e96888886a7456d5d15e2e))


### Bug Fixes

* add -B (foreground) and -d (per-device stats) flags to scrub command ([7c2bdfc](https://github.com/rosavpn/rosadisk-agent/commit/7c2bdfc6f6b2862fe4c85e5ecc7ae9e1149c3e8f))
* add gosec G204 annotations to disk_jobs.go ([e25f317](https://github.com/rosavpn/rosadisk-agent/commit/e25f317d918db075afa4c83307212ef5cf255134))

## [0.14.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.13.0...v0.14.0) (2026-06-16)


### Features

* restructure global config with VolumeJobSchedule and DiskJobSchedule ([1474626](https://github.com/rosavpn/rosadisk-agent/commit/147462635cfd1e218f400b73cc59f2d54f1e6863))
* restructure global config with VolumeJobSchedule and DiskJobSchedule ([0457496](https://github.com/rosavpn/rosadisk-agent/commit/0457496f1bdbdf29a1339bd0ac6df2a9c6b587ac))

## [0.13.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.12.0...v0.13.0) (2026-06-16)


### Features

* add scheduler event emitter for background jobs ([e1cc286](https://github.com/rosavpn/rosadisk-agent/commit/e1cc2866606fe500cf6294fb72aa2eaf938a35cb))
* add scheduler event emitter for background jobs ([5693cd0](https://github.com/rosavpn/rosadisk-agent/commit/5693cd002f9b00675e6d8103affd7ebe4bdb702f))

## [0.12.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.11.0...v0.12.0) (2026-06-15)


### Features

* add global configuration system with time-based job scheduling ([980e0db](https://github.com/rosavpn/rosadisk-agent/commit/980e0db8aaabccaf0f786bffb4ac6d5fbd68e8b2))
* add global configuration system with time-based job scheduling ([98514e0](https://github.com/rosavpn/rosadisk-agent/commit/98514e06926881a45cebdb34e3cc93eb175933aa))


### Bug Fixes

* enable all job schedule options by default ([b342176](https://github.com/rosavpn/rosadisk-agent/commit/b34217601e2d28fb0bc68be6824e590b20656602))

## [0.11.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.10.0...v0.11.0) (2026-06-13)


### Features

* add subvolume endpoints with SQLite state storage ([7405d5e](https://github.com/rosavpn/rosadisk-agent/commit/7405d5e78ab12bb8870decbed28affe2dcd711f4))
* add subvolume endpoints with SQLite state storage ([803dcee](https://github.com/rosavpn/rosadisk-agent/commit/803dcee2604d51d40b23dfe4648f2b58a7451de0))


### Bug Fixes

* add nosec G204 comment to btrfs quota enable command ([df4bd55](https://github.com/rosavpn/rosadisk-agent/commit/df4bd558747d8a1553dc665599b8af3310928324))
* add nosec G204 comments to subvolumes.go exec.Command calls ([a0d24ad](https://github.com/rosavpn/rosadisk-agent/commit/a0d24ad7152471d95fead6bd87507b9aa379f2d7))
* remove unused database/sql import in server_test.go ([c86b23f](https://github.com/rosavpn/rosadisk-agent/commit/c86b23fef4ec3b15d7833b5d9f824639e00132e5))
* respect quota.enabled flag when creating subvolume ([9e7b74d](https://github.com/rosavpn/rosadisk-agent/commit/9e7b74d069c1d0d6ac9b5f8ad9e0027d5e196d2e))
* update server tests for new DB parameter ([8dbc0ee](https://github.com/rosavpn/rosadisk-agent/commit/8dbc0eea76c7d07712a69132772c8faaea195832))

## [0.10.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.9.0...v0.10.0) (2026-06-09)


### Features

* add /v1/mounts endpoints for btrfs mount management ([32e79c8](https://github.com/rosavpn/rosadisk-agent/commit/32e79c8a100ba8d206c46d6bdfc1f26f81363fea))
* add mounts storage layer ([0ddacdd](https://github.com/rosavpn/rosadisk-agent/commit/0ddacdd52f6c37316d1a6267ca41d98ebfd0622c))
* mounts endpoint ([edc740c](https://github.com/rosavpn/rosadisk-agent/commit/edc740c518090e3d6ad674707da0c11032e45ed5))
* update mount response with label and used space ([43fbd0a](https://github.com/rosavpn/rosadisk-agent/commit/43fbd0ad97ec18b3a3f431b39e2f66fa81ce2171))


### Bug Fixes

* add nolint for Sscanf error ([c965369](https://github.com/rosavpn/rosadisk-agent/commit/c9653695ef0fd6c86ffd79886891c4789f36fcf5))
* address gosec warnings in mounts.go ([db1fb4d](https://github.com/rosavpn/rosadisk-agent/commit/db1fb4d44139fed77e67b6efb8f08de65d754e03))
* create mountpoint directory before mounting ([2dd58fb](https://github.com/rosavpn/rosadisk-agent/commit/2dd58fbb3b0769244f1cd0328b2baf06b2f23321))
* get label from btrfs filesystem show ([bbda708](https://github.com/rosavpn/rosadisk-agent/commit/bbda70816b7ac76c6edd126d1783053a825865cb))
* parse label and used from btrfs commands ([c3ef107](https://github.com/rosavpn/rosadisk-agent/commit/c3ef107a5149390ce756deb145d4b308afc2b928))
* use correct gosec nolint format ([e7053c0](https://github.com/rosavpn/rosadisk-agent/commit/e7053c0b679cab1548531ea48deaa5570a53c9da))

## [0.9.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.8.0...v0.9.0) (2026-06-08)


### Features

* add /v1/fs endpoints for btrfs filesystem management ([318a456](https://github.com/rosavpn/rosadisk-agent/commit/318a456fa7b89316193d9db8d21cbc65854b8dde))
* add /v1/fs endpoints for btrfs filesystem management ([1edd99c](https://github.com/rosavpn/rosadisk-agent/commit/1edd99c8345c82955de65ab140b798f00e54ccdb))
* add filesystem creation with validation ([2f9a75b](https://github.com/rosavpn/rosadisk-agent/commit/2f9a75b144a6f312c804277be7c8abb32a59c662))
* add fstype field to disk list endpoint ([c9f7c6f](https://github.com/rosavpn/rosadisk-agent/commit/c9f7c6ff1b7dd960cfe8381f3909eb208961ed43))
* detect RAID profile from btrfs chunk tree ([792e99e](https://github.com/rosavpn/rosadisk-agent/commit/792e99e4353d52cb2d3f0577d99b6f44b6e88273))


### Bug Fixes

* add nosec comments for validated command execution ([f2da478](https://github.com/rosavpn/rosadisk-agent/commit/f2da4780ea1a6986e6e81debe2f56e286367f6bd))
* allow loop devices in path validation ([22ac295](https://github.com/rosavpn/rosadisk-agent/commit/22ac295f3a89708e4101b4266947e54502d591e2))
* include loop devices in disk list ([11f9014](https://github.com/rosavpn/rosadisk-agent/commit/11f901492b66e2fd5ceeef3aa48ee47bbcb07609))
* parse btrfs filesystem details correctly ([2d6cda8](https://github.com/rosavpn/rosadisk-agent/commit/2d6cda86f328018dd162c23c17e6121b38ccc891))
* parse UUID, size and label correctly after filesystem creation ([a1bc954](https://github.com/rosavpn/rosadisk-agent/commit/a1bc9548613786ba7e450945f44d8b73e277046f))
* use minimum device size for RAID1 filesystems ([f28f023](https://github.com/rosavpn/rosadisk-agent/commit/f28f023ad1d63ca9801a282d83873c7936de62e1))
* validate device paths to prevent command injection ([e1546dc](https://github.com/rosavpn/rosadisk-agent/commit/e1546dc726b66a2f6a9df77ba94996a2c4c07c32))

## [0.8.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.7.0...v0.8.0) (2026-06-06)


### Features

* filter disks only and simplify response ([47cbed5](https://github.com/rosavpn/rosadisk-agent/commit/47cbed5aa977b12d5ac40174bc4624355d8852df))
* filter disks only and simplify response ([20b067c](https://github.com/rosavpn/rosadisk-agent/commit/20b067c1b4caf3ed994fbc173dee10d099a0e594))


### Bug Fixes

* update generated code and format files ([c042c90](https://github.com/rosavpn/rosadisk-agent/commit/c042c904723989fe3a13bfb8836725ffdaebd693))

## [0.7.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.6.0...v0.7.0) (2026-06-06)


### Features

* add disk vendor and model fields with parent inheritance ([685c97f](https://github.com/rosavpn/rosadisk-agent/commit/685c97f482fce25a14cc929a613976fb01ba3582))
* add disk vendor and model fields with parent inheritance ([58daef6](https://github.com/rosavpn/rosadisk-agent/commit/58daef6bb4d5d7bac0f88133f28f53ff6fe89b0c))

## [0.6.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.5.0...v0.6.0) (2026-06-06)


### Features

* add disk list endpoint with event-driven architecture ([bbfb054](https://github.com/rosavpn/rosadisk-agent/commit/bbfb054391ec1ec22b58698c1618936c5d20d2e4))
* add disk list endpoint with event-driven architecture ([0b329db](https://github.com/rosavpn/rosadisk-agent/commit/0b329dbb59a741aab72a4db773b8656593f06c04))


### Bug Fixes

* add overflow check for int64 to uint64 conversion ([08dbe20](https://github.com/rosavpn/rosadisk-agent/commit/08dbe209fa187c1b832d998c62fba42144f14deb))

## [0.5.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.4.4...v0.5.0) (2026-06-04)


### Features

* add .opencode/ to .gitignore ([0bf64e4](https://github.com/rosavpn/rosadisk-agent/commit/0bf64e42cec6b115da0c4b71f9b813e6b48eceeb))
* add opencode agentic development configuration ([d4d8225](https://github.com/rosavpn/rosadisk-agent/commit/d4d82256806e344ebc163b0bbdd19a5a6f7c49e5))


### Bug Fixes

* update Go to 1.25.11 to patch security vulnerabilities ([febffe2](https://github.com/rosavpn/rosadisk-agent/commit/febffe222060575e7634bedded1232b422c56e58))

## [0.4.4](https://github.com/rosavpn/rosadisk-agent/compare/v0.4.3...v0.4.4) (2026-05-30)


### Bug Fixes

* use echo for proper newlines in Release file ([8b7f2c1](https://github.com/rosavpn/rosadisk-agent/commit/8b7f2c11f45eb66a7a19f95c283c7b7037e5688d))
* use echo for proper newlines in Release file ([21a4f8f](https://github.com/rosavpn/rosadisk-agent/commit/21a4f8f4bba08ad49c70a2dc272ec911fa048233))

## [0.4.3](https://github.com/rosavpn/rosadisk-agent/compare/v0.4.2...v0.4.3) (2026-05-30)


### Bug Fixes

* generate proper Release file checksum format ([f923772](https://github.com/rosavpn/rosadisk-agent/commit/f9237723408c1ca96797253a22b30537c48d46ba))
* generate proper Release file checksum format ([1431bf8](https://github.com/rosavpn/rosadisk-agent/commit/1431bf82d272419e57c887257f18ec906e2467b2))
* trailing space ([9e48f3e](https://github.com/rosavpn/rosadisk-agent/commit/9e48f3ef185d83d0bb16d05ae7af8f4040e291bf))

## [0.4.2](https://github.com/rosavpn/rosadisk-agent/compare/v0.4.1...v0.4.2) (2026-05-29)


### Bug Fixes

* use direct gh-pages push to bypass environment restrictions ([d4626e8](https://github.com/rosavpn/rosadisk-agent/commit/d4626e8809115499ae230c09e78055fe7b89cf4a))
* use direct gh-pages push to bypass environment restrictions ([0e10045](https://github.com/rosavpn/rosadisk-agent/commit/0e100451680bf0461001356ce6e4683a2335ebcc))

## [0.4.1](https://github.com/rosavpn/rosadisk-agent/compare/v0.4.0...v0.4.1) (2026-05-29)


### Bug Fixes

* correct glob pattern for deb package matching ([a8d731f](https://github.com/rosavpn/rosadisk-agent/commit/a8d731f6d4b312aad2a0be097a10ade2dff7f0cf))
* correct working directory for dpkg-scanpackages ([44544f9](https://github.com/rosavpn/rosadisk-agent/commit/44544f9ae87bc71a992af7aae1a69c5b257d31d0))
* correct working directory for dpkg-scanpackages ([867931f](https://github.com/rosavpn/rosadisk-agent/commit/867931f780e985bfd71972d2a79a5d3341659ad8))

## [0.4.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.3.0...v0.4.0) (2026-05-29)


### Features

* add Debian repository support for GitHub Pages ([30c03af](https://github.com/rosavpn/rosadisk-agent/commit/30c03afd4a2c657a5a3cd16f5e9f38d74eb8dcca))
* add Debian repository support for GitHub Pages ([3be466d](https://github.com/rosavpn/rosadisk-agent/commit/3be466d2197ac45bfe2f98aec3e3d8d2a02ec9c6))

## [0.3.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.2.0...v0.3.0) (2026-05-29)


### Features

* add DEB package building to release workflow ([c94712b](https://github.com/rosavpn/rosadisk-agent/commit/c94712b5361de3f6b9bc5fde3c61224121f43bff))
* add DEB package building to release workflow ([36bf0cb](https://github.com/rosavpn/rosadisk-agent/commit/36bf0cb1e03100d70f84e48ab823180567bfaa71))
* add systemd unit to DEB package for Debian 13 ([e7c753a](https://github.com/rosavpn/rosadisk-agent/commit/e7c753ab361e7673d410f38cc53f9db7bfb97994))

## [0.2.0](https://github.com/rosavpn/rosadisk-agent/compare/v0.1.0...v0.2.0) (2026-05-29)


### Features

* add release-please for automatic versioning and releases ([f9d478b](https://github.com/rosavpn/rosadisk-agent/commit/f9d478bcf62ac4cc9b6a5639dad24c5e1b487306))
* add release-please for automatic versioning and releases ([2282443](https://github.com/rosavpn/rosadisk-agent/commit/22824431941605732bfa53ce8cb15555a02032a5))


### Bug Fixes

* use PAT for release-please to allow PR creation ([de1d4e7](https://github.com/rosavpn/rosadisk-agent/commit/de1d4e717469a896d1115ac472763344d5883a82))
* use PAT for release-please to allow PR creation ([95ee8da](https://github.com/rosavpn/rosadisk-agent/commit/95ee8daa369dbf6b0b126a6b0196126cf66ea7ae))
