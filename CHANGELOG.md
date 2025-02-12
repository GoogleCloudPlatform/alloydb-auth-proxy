# Changelog

## [1.12.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.12.1...v1.12.2) (2025-02-12)


### Bug Fixes

* bump dependencies to latest ([#758](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/758)) ([a45846c](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a45846c2f9cb5aa958e10cdc9072a7db1631334a))

## [1.12.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.12.0...v1.12.1) (2025-01-15)


### Bug Fixes

* bump dependencies to latest ([#747](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/747)) ([be4d719](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/be4d719b9fa4b12024e615b7c9b823303f3be851))

## [1.12.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.11.3...v1.12.0) (2024-12-11)


### Features

* accept GET method at /quitquitquit ([#726](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/726)) ([19b1ec8](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/19b1ec8be37e12cdfdd9b4d16153724b2d02e3d7))

## [1.11.3](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.11.2...v1.11.3) (2024-11-13)


### Bug Fixes

* bump dependencies to latest ([#720](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/720)) ([0ed87a9](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/0ed87a9c940cd9778fc1dcb50733568d31b1c140))

## [1.11.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.11.1...v1.11.2) (2024-10-09)


### Bug Fixes

* bump dependencies to latest ([#711](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/711)) ([2b0e567](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/2b0e567473ba326c6ac9e30ac8c97e848f189769))

## [1.11.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.11.0...v1.11.1) (2024-09-19)


### Bug Fixes

* bump dependencies to latest ([#704](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/704)) ([a41063e](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a41063e58962680454041ad0b88e7d690c5cecbc))

## [1.11.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.10.2...v1.11.0) (2024-08-15)


### Features

* exit when FUSE errors on accept ([#694](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/694)) ([e0bd377](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e0bd3772f3c30cf8a7da14ff940dd57c4a0ca97d))
* replace buster with bookworm container ([#672](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/672)) ([d81b90d](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/d81b90dd0a6cf70e275b47c58178636450c2141f))
* support for exit zero sigterm ([#684](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/684)) ([790b935](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/790b935e29fd687688faf885de3387f136d000a9))
* support for min-sigterm-delay ([#693](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/693)) ([2b0de79](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/2b0de79d5d2f6b5365a3ac023ce5868302fe4127))


### Bug Fixes

* ignore go-fuse ctx in Lookup ([#675](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/675)) ([57d3e80](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/57d3e802cab62ea747b8eb51b292d45402d64f49))

## [1.10.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.10.1...v1.10.2) (2024-07-10)


### Bug Fixes

* update dependencies to latest ([#667](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/667)) ([3f0c143](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/3f0c143de7f50700e1eb42feb7b7aa89601306c7))

## [1.10.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.10.0...v1.10.1) (2024-06-13)


### Bug Fixes

* bump dependencies to latest ([#656](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/656)) ([1b605d6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/1b605d6f85b8f6c6f5ecbaeefebd785c025381a8))

## [1.10.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.9.0...v1.10.0) (2024-05-15)


### Features

* add support for a lazy refresh ([#2184](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/2184)) ([#644](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/644)) ([b375d34](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/b375d34313190fd296010cb175c996d43485ab38)), closes [#625](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/625)
* support static connection info ([#648](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/648)) ([78c3131](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/78c31319f85705d9f3eca4f030522f5088731412))


### Bug Fixes

* don't depend on downstream in readiness check ([#641](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/641)) ([3a7c789](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/3a7c789fd40fe7810b215d6f3b5b9341448800fc))
* ensure graceful shutdown on fuse error ([#638](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/638)) ([eb4435b](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/eb4435bf74018107915d03d250a78f0fe7541ef3))

## [1.9.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.8.0...v1.9.0) (2024-04-17)


### Features

* add support for a config file ([#612](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/612)) ([d9f3845](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/d9f3845e56107bb35684e43cf1889787b37d51be))
* add support for PSC ([#613](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/613)) ([1ef4f60](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/1ef4f60e7e6fad02d84c709b60731b436f80420a))
* use Google managed base images ([#611](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/611)) ([16ed6a8](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/16ed6a8a4822ebce770d74401e5480ff6e70213b))


### Bug Fixes

* switch to public mirrors of base containers ([#627](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/627)) ([c433f43](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/c433f4320a98e8445fd76bf7156a40c3c07b83e8))

## [1.8.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.7.1...v1.8.0) (2024-03-14)


### Features

* add support for debug logs ([#596](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/596)) ([7586c15](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/7586c15b7d7b764806a43c1dc7ef27dc1b70df8a))

## [1.7.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.7.0...v1.7.1) (2024-02-21)


### Bug Fixes

* bump dependencies to latest ([#585](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/585)) ([dce6edd](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/dce6edd4b71f49b9de84df635583ef2d3a6cd34b))

## [1.7.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.6.2...v1.7.0) (2024-01-29)


### Features

* add support for public IP connections ([#566](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/566)) ([ac21696](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/ac216965ef78f89cf1882a3bc822008bcab7e297))

## [1.6.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.6.1...v1.6.2) (2024-01-17)


### Bug Fixes

* update dependencies to latest versions ([#553](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/553)) ([746f8b1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/746f8b1492394f4170885774626d0ed425e8c579))

## [1.6.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.6.0...v1.6.1) (2023-12-13)


### Bug Fixes

* correctly apply container image labels ([#524](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/524)) ([15bbe33](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/15bbe330e8cc42a9101dce72986957494b53c037))

## [1.6.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.5.0...v1.6.0) (2023-12-04)


### Features

* add wait command support ([#512](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/512)) ([a1506d1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a1506d129c4bc4c14884fd1f28f81782ce3e0313)), closes [#511](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/511)

## [1.5.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.4.1...v1.5.0) (2023-11-15)


### Features

* add support for Auto IAM AuthN ([#423](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/423)) ([e854766](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/e85476658496d2f3b2692d2885a02ba925fa0200))

## [1.4.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.4.0...v1.4.1) (2023-10-17)


### Bug Fixes

* bump dependencies to latest ([#478](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/478)) ([3dc8bd2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/3dc8bd29b84ef09328fd647897c2402730954abe))

## [1.4.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.3.2...v1.4.0) (2023-09-19)


### Features

* Add support for systemd notify ([#425](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/425)) ([71b2fae](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/71b2fae7e44f1bd278740789feec094700bbcfe9))
* add Windows service support ([#429](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/429)) ([2ad57d6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/2ad57d6625cd868e7d39b5a0578f3061334a2133))

## [1.3.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.3.1...v1.3.2) (2023-08-16)


### Bug Fixes

* update dependencies to latest ([#414](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/414)) ([bade95a](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/bade95ad5315d5254f3b49ca86ee00ebff3dbecc))

## [1.3.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.3.0...v1.3.1) (2023-07-19)


### Bug Fixes

* update dependencies to latest ([#386](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/386)) ([80763dd](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/80763dd1a25fac2522f701c1d82c36481525579b))

## [1.3.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.2.4...v1.3.0) (2023-06-15)


### Features

* add support for connection test ([#367](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/367)) ([9157991](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/915799102ba8f553bd683cf370f1e2b42e030d10))

## [1.2.4](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.2.3...v1.2.4) (2023-05-19)


### Bug Fixes

* update dependencies to latest versions ([#349](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/349)) ([0fe28a5](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/0fe28a5d429fbf2e3be082579df65499b9288553))

## [1.2.3](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.2.2...v1.2.3) (2023-04-24)


### Bug Fixes

* allow `--structured-logs` and `--quiet` flags together ([#317](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/317)) ([2cba2f0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/2cba2f07767164327f534a1c748e943a2d0d5cc3))
* pass dial options to FUSE mounts ([#319](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/319)) ([65ce745](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/65ce745b0585b7353b1c8bac596032c9c609d416))

## [1.2.2](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.2.1...v1.2.2) (2023-03-22)


### Bug Fixes

* bump deps to latest ([#288](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/288)) ([14e4793](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/14e4793478936b5d1a8b82db0a9bb8ad93db2209))

## [1.2.1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.2.0...v1.2.1) (2023-02-23)


### Bug Fixes

* build statically linked binaries ([#272](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/272)) ([75c05a6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/75c05a606d5ebb675aefe6f2c557082f2ed0e3bc))

## [1.2.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.1.0...v1.2.0) (2023-02-17)


### Features

* add admin server with pprof ([#236](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/236)) ([46d59c8](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/46d59c8b43cb58852b513e2114f9307176b76d0c))
* add support for Go 1.20 ([#256](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/256)) ([1f4f1c7](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/1f4f1c7f1c85d175576189d045c9c58d6c25a3f5))
* add support for min ready instances ([#229](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/229)) ([f9d262a](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/f9d262aa4244bc9be80c096b76378af74722511f))
* add support for quitquitquit endpoint ([#255](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/255)) ([afc7bc1](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/afc7bc1ab620999e52763e723d782d5312c7cf3d))
* add support for unix-socket-path instance parameter. ([#251](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/251)) ([3c17cfa](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/3c17cfa4d7af792a70c6f2f2d8e792fdd4687f26))
* add support for user-agent configuration ([#232](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/232)) ([0b1d3e9](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/0b1d3e962bd0487b045a32777f54709d84ff2bea))


### Bug Fixes

* honor request context in readiness check ([#265](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/265)) ([96ec2aa](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/96ec2aab0a554f7735ab9dbcb084099e7477e714))
* report the real error when newSocketMount() fails ([#252](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/252)) ([d76a334](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/d76a3346307640b5e3a81f3d5bbefcdf55e94e5b))
* return correct exit code on SIGTERM ([#231](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/231)) ([a0ac50c](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a0ac50cf968a77b94959d9dac232a9777391a0b9))
* use correct OAuth2 scope for impersonation ([#241](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/241)) ([af91431](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/af914315111360a963377a3a4802ef612f7013e5))

## [1.1.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v1.0.0...v1.1.0) (2023-01-17)


### Features

* fail on startup if instance uri is invalid ([#210](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/210)) ([49b9efd](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/49b9efdb141c710dae138621b802dd9caf990185))
* use shorter instance uri when logging ([#213](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/213)) ([87c44db](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/87c44db1395ca08af054bdea4ee2b8f3f11e7932))


### Bug Fixes

* correctly apply metadata to user agent ([#218](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/218)) ([a023b45](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/a023b45eae7f49b2ccee0ec6189eed4002af4fb6))

## [1.0.0](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/compare/v0.6.2...v1.0.0) (2022-12-13)


### Features

* add quiet flag ([#196](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/196)) ([c307639](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/c3076397834668e24cf1640a4e8ee294716734ae))
* add support for JSON credentials ([#188](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/188)) ([3347d3b](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/3347d3b33c7177f5f16384e995db2606ecc784e6))
* add support for service account impersonation ([#192](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/192)) ([de15073](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/de1507336b9998283ca7f0918798f48d124a10c4))
* configure the proxy with environment variables ([#197](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/197)) ([cfcf17d](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/cfcf17dc39ca80e441a4211d3a3024ce04429a3f))


### Bug Fixes

* correct error check in check connections ([#195](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/195)) ([8513d07](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/8513d07a75e27c8150d6f707dda2dd60b5a652f4))
* restore openbsd and freebsd support ([#191](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/191)) ([c53f14e](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/c53f14e2f56492d277aa8a2966c20b5455c38042))
* use alloydb-tmp as default tmp dir ([#199](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/199)) ([30587d6](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/30587d610b550404cc3365d6429e943fe16ed358))


### Miscellaneous Chores

* release 1.0.0 ([#208](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/issues/208)) ([f8e557b](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/commit/f8e557b36e72aab967186740c05c77b5eb7d5263))

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
