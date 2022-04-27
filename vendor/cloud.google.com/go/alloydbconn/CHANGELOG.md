# Changelog

## 0.1.0 (2022-04-26)


### ⚠ BREAKING CHANGES

* use instance uri instead of conn name (#15)

### Features

* add AlloyDB refresher ([#2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/2)) ([d0d6a11](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/d0d6a119fcb3cc5613de065a168f415dbce70789))
* add support for dialer ([#4](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/4)) ([483ffda](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/483ffdae1870835db79aa04c59a6322b9ec8e9bb))
* add WithUserAgent opt ([#10](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/10)) ([6582164](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/658216477813b92aadfd44403b9389dcaea9f081))
* switch to prod endpoint ([#13](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/13)) ([b477122](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b47712202088e43533820c51633dff65fe552ce4))
* use v1beta endpoint ([#16](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/16)) ([bfe5fe5](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/bfe5fe56294c76bf7be4ad1ba09cc7b982479d24))


### Bug Fixes

* admin API client handles non-20x responses ([#14](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/14)) ([c2f5dc9](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/c2f5dc92e1a57262c10cd715fc6082a931d0cf70))
* prevent memory leak in driver ([#22](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/22)) ([861d798](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/861d798e0715f16b88d501950a8d9a0493cc8257))
* use instance uri instead of conn name ([#15](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/15)) ([0da01fd](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/0da01fd311f1e8829be0a9eb0efdeb169ee7c555))
