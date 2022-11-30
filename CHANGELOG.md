# Changelog

## [0.6.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.6.1...v0.6.2) (2022-11-30)


### Bug Fixes

* limit ephemeral certificates to 1 hour ([#180](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/180)) ([21932fc](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/21932fc96a1cde8c729087f5b2eec0a955938294))

## [0.6.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.6.0...v0.6.1) (2022-11-15)


### Bug Fixes

* update dependencies to latest versions ([#172](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/172)) ([09252f3](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/09252f3d5418dc43ce71361c14e17f52364a5278))

## [0.6.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.5.1...v0.6.0) (2022-10-18)


### Features

* add bullseye container ([#147](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/147)) ([e9f70c6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e9f70c622f5a0c2d1dde462f8ea44ef6e643fecb))
* add support for FUSE ([#135](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/135)) ([e383f58](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e383f582e9d193e381be407796b7663f9f6adf92)), closes [#132](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/132)
* bump to Go 1.19 ([#140](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/140)) ([773b0b7](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/773b0b79db1f999071ed00c4aee9eeb5af630e0f))


### Bug Fixes

* add entrypoint to Dockerfiles ([#128](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/128)) ([1d03b71](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/1d03b71dd83a01ed4b84376b50345e4afacc0e25))
* add support for legacy project names in FUSE ([#137](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/137)) ([b137ae0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/b137ae02ceb0775f7f87122f8f8e1a9d6a84f113))
* allow group and other access to Unix socket ([#136](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/136)) ([5649176](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/564917604c09a2d06f6a917bf8d5a57754dc91f6))
* support configuration of HTTP server address ([#1365](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/1365)) ([#131](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/131)) ([bd88339](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/bd88339a242e9550b7e601b347b4500892112730))

## [0.5.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.5.0...v0.5.1) (2022-09-07)


### Bug Fixes

* update dependencies to latest versions ([#119](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/119)) ([642f951](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/642f951899e1728c7e824000a038d9c2741879b4))

## [0.5.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.4.0...v0.5.0) (2022-08-02)


### Features

* add support for health-check flag ([#85](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/85)) ([e0b95b9](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e0b95b9a0a6c841190950f36eee61b58abb6e66c))


### Bug Fixes

* make Prometheus namespace optional ([#87](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/87)) ([0090b97](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/0090b977341fd1e7fb3afb58dbe202e6b2863146))

## [0.4.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.3.0...v0.4.0) (2022-07-19)


### Features

* add flag to specify API endpoint  ([#67](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/67)) ([c63b7b4](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/c63b7b475a5b9b76c60c43642d8a6ae441c0ee91))
* add gcloud-auth flag ([#43](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/43)) ([4bfa258](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/4bfa258216f7daa9e7310a28475a628d45333212))
* add max connections flag ([#63](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/63)) ([a5d8ee8](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a5d8ee8ba2f34d8f41f4d972f6513ca9b6091aca))
* add max-sigterm-delay flag ([#64](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/64)) ([2c9864d](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/2c9864d20c01b60d54402dc29d7354d1599f1efa))
* add support for Cloud Monitoring and Cloud Trace ([#60](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/60)) ([b928921](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/b9289214e08b3bbbca1db2fcb7c77156791d23b7))
* add support for Prometheus metrics ([#58](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/58)) ([b6fcbf3](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/b6fcbf32320151aaef282d04b4c8fd1d5e0c9049))
* add support for structured logs ([#66](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/66)) ([5a1e8e8](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/5a1e8e8352b3e8734731adab4c3e57ec92a61e0f))


### Bug Fixes

* resolve data race on closing client ([#62](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/62)) ([e2b40c4](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e2b40c43e3670add413cc540c3be7556a0483e0f))

## [0.3.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.2.0...v0.3.0) (2022-05-31)


### Features

* add support for unix sockets ([#44](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/44)) ([783db6a](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/783db6aba3d408fa57d7b86db895fae1f97583c9))

## [0.2.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.1.0...v0.2.0) (2022-05-19)


### Features

* make Docker images ARM-friendly ([#20](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/20)) ([bc56066](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/bc56066f46e49543f083f634995d12a693423253))


### Bug Fixes

* address alignment for 32-bit binaries ([55247b1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/55247b10ef3215cb5d39a51a3781750bdb164c52))
* specify scope when using --credentials-file flag ([55247b1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/55247b10ef3215cb5d39a51a3781750bdb164c52))

## 0.1.0 (2022-04-27)


### Features

* add Dockerfiles and build config with vendoring ([#3](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/3)) ([273c24f](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/273c24f75d89b15bbe05a5b65ed3d32fa41b7a4b))
* add support for metadata in version ([#6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/6)) ([ca116ec](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/ca116ec0c931a0309ae745e8b102bcd2865468ae))
* bump Go connector to v0.1.0 ([#15](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/15)) ([ce27be6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/ce27be6716ab30cdff8035a89f56f4c68b892643))
* update connector to use v1beta ([#12](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/12)) ([e0dfbdf](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e0dfbdfad13bfe209b929bbe41b1a132cd808348))
* use alloydb-go-connector ([#2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/2)) ([896ba1c](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/896ba1c6dc01991b33dc7624ddda12963661c1b8))
