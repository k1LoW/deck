# Changelog

## [v1.21.6](https://github.com/k1LoW/deck/compare/v1.21.5...v1.21.6) - 2025-10-29
### New Features 🎉
- fix: simplify image comparison logic by removing MIME type check by @k1LoW in https://github.com/k1LoW/deck/pull/461
### Other Changes
- chore(deps): bump google.golang.org/api from 0.249.0 to 0.250.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/455
- chore: setup tagpr labels by @k1LoW in https://github.com/k1LoW/deck/pull/459
- chore(deps): bump the dependencies group across 1 directory with 4 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/458
- chore(deps): bump google.golang.org/api from 0.252.0 to 0.253.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/460

## [v1.21.5](https://github.com/k1LoW/deck/compare/v1.21.4...v1.21.5) - 2025-09-15
### Fix bug 🐛
- [k1LoW/deck#448] fix link text including underscore is partially missing by @takaidohigasi in https://github.com/k1LoW/deck/pull/449
### Other Changes
- chore(deps): bump the dependencies group with 4 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/445
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/444
- ci: set persist-credentials false by @Songmu in https://github.com/k1LoW/deck/pull/447
- chore: add testdata/link.md-1.golden.png by @Songmu in https://github.com/k1LoW/deck/pull/451
- refactoring: merge fragments in md package by @Songmu in https://github.com/k1LoW/deck/pull/452
- chore(deps): bump the dependencies group with 3 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/454
- chore(deps): bump Songmu/tagpr from 1.8.0 to 1.9.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/453

## [v1.21.4](https://github.com/k1LoW/deck/compare/v1.21.3...v1.21.4) - 2025-09-07
### Fix bug 🐛
- fix: when slide pages are added or moved, the freeze option does not behave correctly. by @k1LoW in https://github.com/k1LoW/deck/pull/442

## [v1.21.3](https://github.com/k1LoW/deck/compare/v1.21.2...v1.21.3) - 2025-09-05
### Fix bug 🐛
- hide deprecation warning when using `--base` by @yukukotani in https://github.com/k1LoW/deck/pull/439

## [v1.21.2](https://github.com/k1LoW/deck/compare/v1.21.1...v1.21.2) - 2025-09-05
### Other Changes
- doc: refine README.md by @Songmu in https://github.com/k1LoW/deck/pull/435
- feat: stop forcing command execution through bash by @Songmu in https://github.com/k1LoW/deck/pull/437
- ci: use use_existing_draft instead of replace_existing_draft by @Songmu in https://github.com/k1LoW/deck/pull/438

## [v1.21.1](https://github.com/k1LoW/deck/compare/v1.21.0...v1.21.1) - 2025-09-04
### Other Changes
- docs: replace deprecated --from flag with --base flag in README by @7kaji in https://github.com/k1LoW/deck/pull/430
- chore: move unused public methods from the deck command to export.go by @Songmu in https://github.com/k1LoW/deck/pull/432
- chore: remove redundant checks by @Songmu in https://github.com/k1LoW/deck/pull/433
- chore: enhance logs in applying by @Songmu in https://github.com/k1LoW/deck/pull/434

## [v1.21.0](https://github.com/k1LoW/deck/compare/v1.20.0...v1.21.0) - 2025-09-03
### New Features 🎉
- feat: support strikethrough GFM syntax ('~~') by @Songmu in https://github.com/k1LoW/deck/pull/428
### Other Changes
- chore: use cmd.Print by @Songmu in https://github.com/k1LoW/deck/pull/426
- doc: comprehensive documentation for supported markdown by @Songmu in https://github.com/k1LoW/deck/pull/429

## [v1.20.0](https://github.com/k1LoW/deck/compare/v1.19.2...v1.20.0) - 2025-09-02
### New Features 🎉
- feat: implement `deck doctor` command for environment verification by @Songmu in https://github.com/k1LoW/deck/pull/424

## [v1.19.2](https://github.com/k1LoW/deck/compare/v1.19.1...v1.19.2) - 2025-09-01
### Other Changes
- chore: separate out func handleBlockquote by @Songmu in https://github.com/k1LoW/deck/pull/418
- chore(deps): bump github.com/google/cel-go from 0.26.0 to 0.26.1 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/421
- chore(deps): bump google-github-actions/auth from 2.1.12 to 3.0.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/420
- refactoring: adjust Image cache mechanism by @Songmu in https://github.com/k1LoW/deck/pull/422
- chore: return error in uploading images for failfast by @Songmu in https://github.com/k1LoW/deck/pull/423

## [v1.19.1](https://github.com/k1LoW/deck/compare/v1.19.0...v1.19.1) - 2025-08-31
### Other Changes
- chore: define isPublicURL to determine whether the image is to be upload or not by @Songmu in https://github.com/k1LoW/deck/pull/416

## [v1.19.0](https://github.com/k1LoW/deck/compare/v1.18.1...v1.19.0) - 2025-08-30
### New Features 🎉
- feat: support image link by @Songmu in https://github.com/k1LoW/deck/pull/414

## [v1.18.1](https://github.com/k1LoW/deck/compare/v1.18.0...v1.18.1) - 2025-08-30
### New Features 🎉
- feat: add support for normalizing CRLF line endings by @k1LoW in https://github.com/k1LoW/deck/pull/413
### Other Changes
- chore: use pull_request_target to run integration test on forked repo by @Songmu in https://github.com/k1LoW/deck/pull/410

## [v1.18.0](https://github.com/k1LoW/deck/compare/v1.17.3...v1.18.0) - 2025-08-30
### New Features 🎉
- feat: add --base flag to new subcommand by @Songmu in https://github.com/k1LoW/deck/pull/409
### Other Changes
- refactoring: add actionTypeSentinel to simplify code by @Songmu in https://github.com/k1LoW/deck/pull/405
- refactoring: add a deck.fresh flag to reduce unnecessary requests by @Songmu in https://github.com/k1LoW/deck/pull/407
- doc: trouble shooting about a setting permission for images by @Songmu in https://github.com/k1LoW/deck/pull/408

## [v1.17.3](https://github.com/k1LoW/deck/compare/v1.17.2...v1.17.3) - 2025-08-28
### Fix bug 🐛
- fix: If a line contains only a single `<br>`, it will not be broken and will be escaped. by @k1LoW in https://github.com/k1LoW/deck/pull/402
### Other Changes
- feat: allow page delimiter with trailing spaces or tabs by @k1LoW in https://github.com/k1LoW/deck/pull/403

## [v1.17.2](https://github.com/k1LoW/deck/compare/v1.17.1...v1.17.2) - 2025-08-25
### Other Changes
- chore(deps): bump google.golang.org/api from 0.247.0 to 0.248.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/398
- chore: guide users to doc when HTTP client is not constructed by @exoego in https://github.com/k1LoW/deck/pull/387
- chore: use cmd.Print instead of fmt.Fprint for messages by @Songmu in https://github.com/k1LoW/deck/pull/400

## [v1.17.1](https://github.com/k1LoW/deck/compare/v1.17.0...v1.17.1) - 2025-08-24
### Fix bug 🐛
- fix: paragraph handling in blockquotes by @Songmu in https://github.com/k1LoW/deck/pull/396

## [v1.17.0](https://github.com/k1LoW/deck/compare/v1.16.1...v1.17.0) - 2025-08-24
### New Features 🎉
- feat: derive the PDF name to be exported from the MD file name by @Songmu in https://github.com/k1LoW/deck/pull/392
- feat: reuse blockquote block if possible by @Songmu in https://github.com/k1LoW/deck/pull/394
### Other Changes
- doc: quick links to GCP configuration by @exoego in https://github.com/k1LoW/deck/pull/388

## [v1.16.1](https://github.com/k1LoW/deck/compare/v1.16.0...v1.16.1) - 2025-08-20
### Other Changes
- perf: for public image URLs, skip upload images via Google Drive. by @Songmu in https://github.com/k1LoW/deck/pull/385

## [v1.16.0](https://github.com/k1LoW/deck/compare/v1.15.3...v1.16.0) - 2025-08-20
### New Features 🎉
- feat: support Backslash escapes by @Songmu in https://github.com/k1LoW/deck/pull/382

## [v1.15.3](https://github.com/k1LoW/deck/compare/v1.15.2...v1.15.3) - 2025-08-20
### Fix bug 🐛
- fix: removal of unnecessary <br> processing by @Songmu in https://github.com/k1LoW/deck/pull/379
### Other Changes
- ci: make tagpr work with workflow dispatch by @Songmu in https://github.com/k1LoW/deck/pull/381

## [v1.15.2](https://github.com/k1LoW/deck/compare/v1.15.1...v1.15.2) - 2025-08-19
### Fix bug 🐛
- fix: correct handling of `ignore` outside in non-watch mode. by @k1LoW in https://github.com/k1LoW/deck/pull/378
### Other Changes
- Fix typo in README.md by @willnet in https://github.com/k1LoW/deck/pull/376
- chore: refine around logs by @Songmu in https://github.com/k1LoW/deck/pull/377

## [v1.15.1](https://github.com/k1LoW/deck/compare/v1.15.0...v1.15.1) - 2025-08-18
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/370
- chore(deps): bump the dependencies group with 4 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/369
- fix: resolving inconsistencies in the information structure of list bullets by @Songmu in https://github.com/k1LoW/deck/pull/374

## [v1.15.0](https://github.com/k1LoW/deck/compare/v1.14.0...v1.15.0) - 2025-08-16
### New Features 🎉
- feat: Add basePresentationID configuration for default theme reuse by @Songmu in https://github.com/k1LoW/deck/pull/367
- feat: Add JSON schema for configuration validation by @Songmu in https://github.com/k1LoW/deck/pull/368
### Other Changes
- ci: introduce integration test on GitHub Actions by @Songmu in https://github.com/k1LoW/deck/pull/355
- ci: separate octocov settings for full tests and standard tests by @Songmu in https://github.com/k1LoW/deck/pull/366

## [v1.14.0](https://github.com/k1LoW/deck/compare/v1.13.1...v1.14.0) - 2025-08-15
### New Features 🎉
- feat: support service account with several authentication methods by @Songmu in https://github.com/k1LoW/deck/pull/359
### Other Changes
- chore: define deck.newDeck for code reuse by @Songmu in https://github.com/k1LoW/deck/pull/360
- chore: update and cleanup Makefile by @Songmu in https://github.com/k1LoW/deck/pull/362
- chore: enhance and filter error.json by @Songmu in https://github.com/k1LoW/deck/pull/364

## [v1.13.1](https://github.com/k1LoW/deck/compare/v1.13.0...v1.13.1) - 2025-08-14
### Fix bug 🐛
- fix: to work properly with Shared Drive by @Songmu in https://github.com/k1LoW/deck/pull/356
### Other Changes
- chore: Define BuildTestOptions for debugging on testing by @Songmu in https://github.com/k1LoW/deck/pull/358

## [v1.13.0](https://github.com/k1LoW/deck/compare/v1.12.1...v1.13.0) - 2025-08-13
### Other Changes
- chore: improv the efficiency of screenshot capture during testing by @Songmu in https://github.com/k1LoW/deck/pull/351
- doc: description of integration with laminate by @Songmu in https://github.com/k1LoW/deck/pull/352
- feat: add --folder-id flag and config for Drive folder by @Songmu in https://github.com/k1LoW/deck/pull/354

## [v1.12.1](https://github.com/k1LoW/deck/compare/v1.12.0...v1.12.1) - 2025-08-12
### Fix bug 🐛
- fix: correct `ignore` handing by @k1LoW in https://github.com/k1LoW/deck/pull/349

## [v1.12.0](https://github.com/k1LoW/deck/compare/v1.11.0...v1.12.0) - 2025-08-12
### Other Changes
- chore: cleanup code and docs by @Songmu in https://github.com/k1LoW/deck/pull/342
- chore: define d.getRequestForStyle for commonization and use it by @Songmu in https://github.com/k1LoW/deck/pull/344
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/345
- feat: support debug level logging with -vv option by @Songmu in https://github.com/k1LoW/deck/pull/346
- chore: use (strings|bytes).SplitSeq if we can by @Songmu in https://github.com/k1LoW/deck/pull/347
- chore: display detailed API error message on debug logging by @Songmu in https://github.com/k1LoW/deck/pull/348

## [v1.11.0](https://github.com/k1LoW/deck/compare/v1.10.1...v1.11.0) - 2025-08-10
### Fix bug 🐛
- fix: adjust configuration and data file placement rules by @Songmu in https://github.com/k1LoW/deck/pull/340

## [v1.10.1](https://github.com/k1LoW/deck/compare/v1.10.0...v1.10.1) - 2025-08-09
### Other Changes
- fix: issue where all pages are listed as diff pages in DiffContents due to the `defaults` setting by @Songmu in https://github.com/k1LoW/deck/pull/338

## [v1.10.0](https://github.com/k1LoW/deck/compare/v1.9.0...v1.10.0) - 2025-08-08
### Other Changes
- chore: tidy up ApplyPage by @Songmu in https://github.com/k1LoW/deck/pull/332
- feat: GFM table syntax support by @Songmu in https://github.com/k1LoW/deck/pull/335

## [v1.9.0](https://github.com/k1LoW/deck/compare/v1.8.0...v1.9.0) - 2025-08-04
### New Features 🎉
- feat: support shared drive (always enable) by @takaidohigasi in https://github.com/k1LoW/deck/pull/330
- feat: add support for code block to image conversion command in frontmatter or config by @k1LoW in https://github.com/k1LoW/deck/pull/331
### Other Changes
- chore(deps): bump google.golang.org/api from 0.243.0 to 0.244.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/327

## [v1.8.0](https://github.com/k1LoW/deck/compare/v1.7.0...v1.8.0) - 2025-08-03
### New Features 🎉
- feat: support config.yml for global setting by @k1LoW in https://github.com/k1LoW/deck/pull/326
### Fix bug 🐛
- fix: correct profile name validation by @k1LoW in https://github.com/k1LoW/deck/pull/325
### Other Changes
- chore(deps): bump the dependencies group with 3 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/321
- Remove redundant type cast by @exoego in https://github.com/k1LoW/deck/pull/323
- Remove unused uploadedID in image.SetUploadResult by @exoego in https://github.com/k1LoW/deck/pull/324

## [v1.7.0](https://github.com/k1LoW/deck/compare/v1.6.0...v1.7.0) - 2025-07-27
### New Features 🎉
- feat: adjust imageReplaceMethod for code blocks by @Songmu in https://github.com/k1LoW/deck/pull/317
### Other Changes
- chore: add FUNDING.yml by @Songmu in https://github.com/k1LoW/deck/pull/318
- refactoring: compare.go by @Songmu in https://github.com/k1LoW/deck/pull/319

## [v1.6.0](https://github.com/k1LoW/deck/compare/v1.5.0...v1.6.0) - 2025-07-27
### New Features 🎉
- perf: Apply slides in bulk by @Songmu in https://github.com/k1LoW/deck/pull/316
### Other Changes
- chore: reduce flaky tests by @Songmu in https://github.com/k1LoW/deck/pull/313
- chore: fix flaky again by @Songmu in https://github.com/k1LoW/deck/pull/315

## [v1.5.0](https://github.com/k1LoW/deck/compare/v1.4.1...v1.5.0) - 2025-07-27
### New Features 🎉
- perf: Refine page append processing by @Songmu in https://github.com/k1LoW/deck/pull/310
- feat: Add default definitions for HTML inline styles by @Songmu in https://github.com/k1LoW/deck/pull/312
### Fix bug 🐛
- bugfix: Fixed an issue where images were not displayed correctly when changing layouts by @Songmu in https://github.com/k1LoW/deck/pull/309

## [v1.4.1](https://github.com/k1LoW/deck/compare/v1.4.0...v1.4.1) - 2025-07-26
### New Features 🎉
- perf: Batch delete pages by @Songmu in https://github.com/k1LoW/deck/pull/307

## [v1.4.0](https://github.com/k1LoW/deck/compare/v1.3.1...v1.4.0) - 2025-07-25
### New Features 🎉
- Enhance page separator rules and support HR elements as body separators by @Songmu in https://github.com/k1LoW/deck/pull/305

## [v1.3.1](https://github.com/k1LoW/deck/compare/v1.3.0...v1.3.1) - 2025-07-25
### New Features 🎉
- fix: improve markdown page splitting to respect various markdown syntax by @Songmu in https://github.com/k1LoW/deck/pull/303

## [v1.3.0](https://github.com/k1LoW/deck/compare/v1.2.1...v1.3.0) - 2025-07-23
### New Features 🎉
- feat: add 'open' subcommand to open Google Slides in browser by @Songmu in https://github.com/k1LoW/deck/pull/300

## [v1.2.1](https://github.com/k1LoW/deck/compare/v1.2.0...v1.2.1) - 2025-07-22

## [v1.2.0](https://github.com/k1LoW/deck/compare/v1.1.0...v1.2.0) - 2025-07-22
### New Features 🎉
- feat: add support for user profiles in deck operations by @k1LoW in https://github.com/k1LoW/deck/pull/297

## [v1.1.0](https://github.com/k1LoW/deck/compare/v1.0.0...v1.1.0) - 2025-07-21
### New Features 🎉
- feat(md): replace with CEL-based template expansion by @k1LoW in https://github.com/k1LoW/deck/pull/295

## [v1.0.0](https://github.com/k1LoW/deck/compare/v0.44.0...v1.0.0) - 2025-07-21
### New Features 🎉
- feat(md): add support for default conditions for layout in Frontmatter by @k1LoW in https://github.com/k1LoW/deck/pull/264
- perf: Efficiency and speed improvement of image upload to slides by @Songmu in https://github.com/k1LoW/deck/pull/290
- feat: support `skip` `ignore` `freeze` for `defaults:` by @k1LoW in https://github.com/k1LoW/deck/pull/293
### Fix bug 🐛
- fix: apply by @Songmu in https://github.com/k1LoW/deck/pull/285
- fix(md): handle nil Frontmatter in ToSlides method by @k1LoW in https://github.com/k1LoW/deck/pull/291
- fix: page count by @k1LoW in https://github.com/k1LoW/deck/pull/292
### Other Changes
- chore: separate source files by @Songmu in https://github.com/k1LoW/deck/pull/281
- fix: add empty string check before copying image by @Songmu in https://github.com/k1LoW/deck/pull/287
- chore(deps): bump google.golang.org/api from 0.241.0 to 0.242.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/289
- chore: use slices and cmp package to sort by @Songmu in https://github.com/k1LoW/deck/pull/288
- Release for v1.0.0 by @github-actions[bot] in https://github.com/k1LoW/deck/pull/286

## [v1.0.0](https://github.com/k1LoW/deck/compare/v0.44.0...v1.0.0) - 2025-07-21
### New Features 🎉
- feat(md): add support for default conditions for layout in Frontmatter by @k1LoW in https://github.com/k1LoW/deck/pull/264
- perf: Efficiency and speed improvement of image upload to slides by @Songmu in https://github.com/k1LoW/deck/pull/290
- feat: support `skip` `ignore` `freeze` for `defaults:` by @k1LoW in https://github.com/k1LoW/deck/pull/293
### Fix bug 🐛
- fix: apply by @Songmu in https://github.com/k1LoW/deck/pull/285
- fix(md): handle nil Frontmatter in ToSlides method by @k1LoW in https://github.com/k1LoW/deck/pull/291
- fix: page count by @k1LoW in https://github.com/k1LoW/deck/pull/292
### Other Changes
- chore: separate source files by @Songmu in https://github.com/k1LoW/deck/pull/281
- fix: add empty string check before copying image by @Songmu in https://github.com/k1LoW/deck/pull/287
- chore(deps): bump google.golang.org/api from 0.241.0 to 0.242.0 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/deck/pull/289
- chore: use slices and cmp package to sort by @Songmu in https://github.com/k1LoW/deck/pull/288

## [v0.44.0](https://github.com/k1LoW/deck/compare/v0.43.1...v0.44.0) - 2025-07-19
### New Features 🎉
- chore: Execute integration tests in parallel in groups of three by @Songmu in https://github.com/k1LoW/deck/pull/279
- feat: apply inline styles to headings by @Songmu in https://github.com/k1LoW/deck/pull/273
### Other Changes
- chore: README.md by @Songmu in https://github.com/k1LoW/deck/pull/280

## [v0.43.1](https://github.com/k1LoW/deck/compare/v0.43.0...v0.43.1) - 2025-07-18
### Fix bug 🐛
- fix(compare): ensure images are sorted without modifying input slices by @k1LoW in https://github.com/k1LoW/deck/pull/276
### Other Changes
- fix refresh; search defaultLayout again when disappering by @Songmu in https://github.com/k1LoW/deck/pull/271
- fix clearPlaceholder by @Songmu in https://github.com/k1LoW/deck/pull/270
- feat: define `md.Content.Headings` by @Songmu in https://github.com/k1LoW/deck/pull/258
- chore: merge clear placeholder requests into batchUpdate request by @Songmu in https://github.com/k1LoW/deck/pull/275
- fix: support for different number of images and placeholders / use replace image api by @k1LoW in https://github.com/k1LoW/deck/pull/272

## [v0.43.0](https://github.com/k1LoW/deck/compare/v0.42.2...v0.43.0) - 2025-07-18
### New Features 🎉
- feat: support image placeholder by @Songmu in https://github.com/k1LoW/deck/pull/269

## [v0.42.2](https://github.com/k1LoW/deck/compare/v0.42.1...v0.42.2) - 2025-07-17
### Fix bug 🐛
- fix: improve split markdown as contents by @k1LoW in https://github.com/k1LoW/deck/pull/267
- fix case of presentationID by @Songmu in https://github.com/k1LoW/deck/pull/266

## [v0.42.1](https://github.com/k1LoW/deck/compare/v0.42.0...v0.42.1) - 2025-07-16
### Fix bug 🐛
- fix: proper inline styling in lists by @Songmu in https://github.com/k1LoW/deck/pull/261

## [v0.42.0](https://github.com/k1LoW/deck/compare/v0.41.0...v0.42.0) - 2025-07-15
### New Features 🎉
- feat: HTML element styling feature by @Songmu in https://github.com/k1LoW/deck/pull/225

## [v0.41.0](https://github.com/k1LoW/deck/compare/v0.40.2...v0.41.0) - 2025-07-15
### New Features 🎉
- feat: support HardLineBreak and remove `deck.Fragment.SoftLineBreak` field by @Songmu in https://github.com/k1LoW/deck/pull/247
- feat: support proper paragraph separation in slide by @Songmu in https://github.com/k1LoW/deck/pull/251
- feat: Add breaks frontmatter option to control line break rendering by @Songmu in https://github.com/k1LoW/deck/pull/254
- chore: Merge UpdateTextStyleRequest into one by @Songmu in https://github.com/k1LoW/deck/pull/256
### Fix bug 🐛
- fix: does not copy images/shapes from markdown when update layout by @k1LoW in https://github.com/k1LoW/deck/pull/253
### Other Changes
- chore: use slices package to simplify code by @Songmu in https://github.com/k1LoW/deck/pull/249
- chore: add client.go and move http client related code into it by @Songmu in https://github.com/k1LoW/deck/pull/257

## [v0.40.2](https://github.com/k1LoW/deck/compare/v0.40.1...v0.40.2) - 2025-07-14
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/k1LoW/deck/pull/245

## [v0.40.1](https://github.com/k1LoW/deck/compare/v0.40.0...v0.40.1) - 2025-07-13
### Fix bug 🐛
- fix(md): resolve concurrency issue in code block image generation by @k1LoW in https://github.com/k1LoW/deck/pull/244
### Other Changes
- feat(compare): add sorting logic for BlockQuotes in blockQuotesEqual function by @k1LoW in https://github.com/k1LoW/deck/pull/240
- fix(compare): improve paragraph equality logic / reduce testdata(markdown pages) by @k1LoW in https://github.com/k1LoW/deck/pull/243

## [v0.40.0](https://github.com/k1LoW/deck/compare/v0.39.0...v0.40.0) - 2025-07-13
### New Features 🎉
- feat: support block quotes by @k1LoW in https://github.com/k1LoW/deck/pull/208
- fix(compare): move comparison logic to a new file and simplify equality check by @k1LoW in https://github.com/k1LoW/deck/pull/238
- feat(deck): add affine transform for images and block quotes by @k1LoW in https://github.com/k1LoW/deck/pull/239
### Other Changes
- test: improve integration test by @k1LoW in https://github.com/k1LoW/deck/pull/232
- chore: add linter `funcorder` by @k1LoW in https://github.com/k1LoW/deck/pull/235
- chore: `define (*Deck) getInlineStyleElements(*deck.Fragment)` and refactoring by @Songmu in https://github.com/k1LoW/deck/pull/236

## [v0.39.0](https://github.com/k1LoW/deck/compare/v0.38.0...v0.39.0) - 2025-07-10
### New Features 🎉
- perf: add debounce function to avoid busy loop while watching by @Songmu in https://github.com/k1LoW/deck/pull/227
### Other Changes
- chore: fix lint errors and fix reviewdog/action-golangci-lint by @Songmu in https://github.com/k1LoW/deck/pull/229
- chore: set User-Agent to HTTP Request for crawling manner by @Songmu in https://github.com/k1LoW/deck/pull/228

## [v0.38.0](https://github.com/k1LoW/deck/compare/v0.37.1...v0.38.0) - 2025-07-09
### New Features 🎉
- feat(cmd): add markdown file support to deck new command by @Songmu in https://github.com/k1LoW/deck/pull/223
- perf: optimize watch process by @Songmu in https://github.com/k1LoW/deck/pull/226

## [v0.37.1](https://github.com/k1LoW/deck/compare/v0.37.0...v0.37.1) - 2025-07-08
### Fix bug 🐛
- fix: improve single-character list item rendering by @Songmu in https://github.com/k1LoW/deck/pull/220

## [v0.37.0](https://github.com/k1LoW/deck/compare/v0.36.0...v0.37.0) - 2025-07-07
### New Features 🎉
- feat: add support for skipping slides by @k1LoW in https://github.com/k1LoW/deck/pull/214
### Other Changes
- fix: rename JSON field names to use snake_case by @k1LoW in https://github.com/k1LoW/deck/pull/216

## [v0.36.0](https://github.com/k1LoW/deck/compare/v0.35.0...v0.36.0) - 2025-07-07
### New Features 🎉
- feat: Add ignore page functionality with page configuration comment by @Songmu in https://github.com/k1LoW/deck/pull/213
### Other Changes
- feat(md): add support for block quotes by @k1LoW in https://github.com/k1LoW/deck/pull/206
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/209

## [v0.35.0](https://github.com/k1LoW/deck/compare/v0.34.0...v0.35.0) - 2025-07-03
### New Features 🎉
- feat: Add {{output}} template variable support for code-block-to-image-command by @Songmu in https://github.com/k1LoW/deck/pull/204

## [v0.34.0](https://github.com/k1LoW/deck/compare/v0.33.0...v0.34.0) - 2025-07-02
### New Features 🎉
- make heading inside body bold; it's natural by @Songmu in https://github.com/k1LoW/deck/pull/196
### Other Changes
- doc: use camelCase for frontmatter fields by @Songmu in https://github.com/k1LoW/deck/pull/201

## [v0.33.0](https://github.com/k1LoW/deck/compare/v0.32.0...v0.33.0) - 2025-07-01
### New Features 🎉
- fix: refactor image handling for improved performance and flexibility by @k1LoW in https://github.com/k1LoW/deck/pull/191
- feat(apply): support setting title from frontmatter by @k1LoW in https://github.com/k1LoW/deck/pull/192
### Other Changes
- feat(cmd): add --presentation-id flag and deprecate positional arguments by @k1LoW in https://github.com/k1LoW/deck/pull/189

## [v0.32.0](https://github.com/k1LoW/deck/compare/v0.31.2...v0.32.0) - 2025-07-01
### New Features 🎉
- feat: add angle bracket autolinks support by @Songmu in https://github.com/k1LoW/deck/pull/185
- feat: support dynamic heading levels for title detection by @Songmu in https://github.com/k1LoW/deck/pull/188

## [v0.31.2](https://github.com/k1LoW/deck/compare/v0.31.1...v0.31.2) - 2025-07-01
### Fix bug 🐛
- fix: add CODEBLOCK_CONTENT env to cmd by @Songmu in https://github.com/k1LoW/deck/pull/181

## [v0.31.1](https://github.com/k1LoW/deck/compare/v0.31.0...v0.31.1) - 2025-07-01
### Fix bug 🐛
- fix: check sum comparison of images by @Songmu in https://github.com/k1LoW/deck/pull/179

## [v0.31.0](https://github.com/k1LoW/deck/compare/v0.30.0...v0.31.0) - 2025-06-30
### New Features 🎉
- feat: add YAML frontmatter support with new md.MD type by @Songmu in https://github.com/k1LoW/deck/pull/177
- feat: add presentationID support in frontmatter for simplified apply command by @Songmu in https://github.com/k1LoW/deck/pull/178

## [v0.30.0](https://github.com/k1LoW/deck/compare/v0.29.1...v0.30.0) - 2025-06-30
### New Features 🎉
- feat: create error.json if deck fails by @k1LoW in https://github.com/k1LoW/deck/pull/172
- feat: integrate tail-based logging and enhance error reporting by @k1LoW in https://github.com/k1LoW/deck/pull/174

## [v0.29.1](https://github.com/k1LoW/deck/compare/v0.29.0...v0.29.1) - 2025-06-30
### Other Changes
- refactor(deck): remove redundant DeletePageAfter call in ApplyPages function by @k1LoW in https://github.com/k1LoW/deck/pull/166
- refactor: prepend identifiers with type in object ID generation by @k1LoW in https://github.com/k1LoW/deck/pull/167
- feat(logger): add red color for failure messages by @k1LoW in https://github.com/k1LoW/deck/pull/169
- feat(apply): switch to JSON logging when `--verbose` and enhance action details structure by @k1LoW in https://github.com/k1LoW/deck/pull/170
- chore(deps): bump google.golang.org/api from 0.238.0 to 0.239.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/171

## [v0.29.0](https://github.com/k1LoW/deck/compare/v0.28.0...v0.29.0) - 2025-06-29
### New Features 🎉
- fix: prune old images via markdown by @k1LoW in https://github.com/k1LoW/deck/pull/165
### Fix bug 🐛
- fix: optimize markdown content comparisons by @k1LoW in https://github.com/k1LoW/deck/pull/162
- fix: respond when images are manually deleted in Google Slides by @k1LoW in https://github.com/k1LoW/deck/pull/164

## [v0.28.0](https://github.com/k1LoW/deck/compare/v0.27.1...v0.28.0) - 2025-06-29
### New Features 🎉
- feat: add global image caching mechanism by @k1LoW in https://github.com/k1LoW/deck/pull/160

## [v0.27.1](https://github.com/k1LoW/deck/compare/v0.27.0...v0.27.1) - 2025-06-29
### New Features 🎉
- fix: cache calculated checksums and pHash values for image comparison by @k1LoW in https://github.com/k1LoW/deck/pull/157
### Fix bug 🐛
- fix(md): correct image copying logic in ToSlides by @k1LoW in https://github.com/k1LoW/deck/pull/159

## [v0.27.0](https://github.com/k1LoW/deck/compare/v0.26.0...v0.27.0) - 2025-06-28
### New Features 🎉
- feat(dot): display a spinner when retrying a Google Slides API call by @k1LoW in https://github.com/k1LoW/deck/pull/151
- feat(logger): enhance dot handler for deleted/moved pages by @k1LoW in https://github.com/k1LoW/deck/pull/154
### Other Changes
- fix(deck): rename methods and add logging for page operations by @k1LoW in https://github.com/k1LoW/deck/pull/150
- fix: rename package by @k1LoW in https://github.com/k1LoW/deck/pull/152
- fix(deck): correct log levels for debug and warn messages by @k1LoW in https://github.com/k1LoW/deck/pull/153
- docs: correct `-l` option usage by @lacolaco in https://github.com/k1LoW/deck/pull/155
- feat(dot): enhance spinner handling by @k1LoW in https://github.com/k1LoW/deck/pull/156

## [v0.26.0](https://github.com/k1LoW/deck/compare/v0.25.0...v0.26.0) - 2025-06-26
### New Features 🎉
- feat: support bullet in shape by @k1LoW in https://github.com/k1LoW/deck/pull/147

## [v0.25.0](https://github.com/k1LoW/deck/compare/v0.24.2...v0.25.0) - 2025-06-24
### New Features 🎉
- feat(deck): add support for copying shapes between slides (create/delete) by @k1LoW in https://github.com/k1LoW/deck/pull/145
### Other Changes
- fix(md): rename `value` to `content` in code block by @k1LoW in https://github.com/k1LoW/deck/pull/144

## [v0.24.2](https://github.com/k1LoW/deck/compare/v0.24.1...v0.24.2) - 2025-06-23
### Fix bug 🐛
- fix(deck): correct indentation handling for nested paragraphs by @k1LoW in https://github.com/k1LoW/deck/pull/142

## [v0.24.1](https://github.com/k1LoW/deck/compare/v0.24.0...v0.24.1) - 2025-06-23
### New Features 🎉
- feat(deck): add functionality to copy images between (create/delete) slides when update by @k1LoW in https://github.com/k1LoW/deck/pull/139
### Fix bug 🐛
- fix(md): correct nested list parsing by @k1LoW in https://github.com/k1LoW/deck/pull/140
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/137

## [v0.24.0](https://github.com/k1LoW/deck/compare/v0.23.0...v0.24.0) - 2025-06-22
### New Features 🎉
- feat(md): add support for code blocks in slide content by @k1LoW in https://github.com/k1LoW/deck/pull/129
- feat: add support for converting code blocks to images by @k1LoW in https://github.com/k1LoW/deck/pull/133
### Other Changes
- fix(md): introduce Parser struct for markdown parsing by @k1LoW in https://github.com/k1LoW/deck/pull/131
- Revert "fix(md): introduce Parser struct for markdown parsing" by @k1LoW in https://github.com/k1LoW/deck/pull/132
- fix(deck): simplify default*Layout logic by @k1LoW in https://github.com/k1LoW/deck/pull/134
- fix(md): use context with command by @k1LoW in https://github.com/k1LoW/deck/pull/135

## [v0.23.0](https://github.com/k1LoW/deck/compare/v0.22.1...v0.23.0) - 2025-06-21
### New Features 🎉
- feat(md): support image by @k1LoW in https://github.com/k1LoW/deck/pull/126
- feat(deck): support applying image ( `![img](path/to/image.png)` ) by @k1LoW in https://github.com/k1LoW/deck/pull/128

## [v0.22.1](https://github.com/k1LoW/deck/compare/v0.22.0...v0.22.1) - 2025-06-16
### Fix bug 🐛
- fix(deck): handle excess slides in ApplyPages by @k1LoW in https://github.com/k1LoW/deck/pull/124
### Other Changes
- chore(deps): bump google.golang.org/api from 0.236.0 to 0.237.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/123

## [v0.22.0](https://github.com/k1LoW/deck/compare/v0.21.1...v0.22.0) - 2025-06-15
### New Features 🎉
- fix: improve actions generation logic by @k1LoW in https://github.com/k1LoW/deck/pull/118
### Fix bug 🐛
- fix: improve action generation and similarity scoring by @k1LoW in https://github.com/k1LoW/deck/pull/121
### Other Changes
- test: add FuzzGenerateActions by @k1LoW in https://github.com/k1LoW/deck/pull/120
- test: add fuzzing workflow by @k1LoW in https://github.com/k1LoW/deck/pull/122

## [v0.21.1](https://github.com/k1LoW/deck/compare/v0.21.0...v0.21.1) - 2025-06-15
### Fix bug 🐛
- fix: resolve duplicate output in markdown lists separated by blank lines by @hanhan1978 in https://github.com/k1LoW/deck/pull/116
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/113

## [v0.21.0](https://github.com/k1LoW/deck/compare/v0.20.0...v0.21.0) - 2025-06-12
### New Features 🎉
- fix: improve file change detection for vim compatibility by @hanhan1978 in https://github.com/k1LoW/deck/pull/114

## [v0.20.0](https://github.com/k1LoW/deck/compare/v0.19.1...v0.20.0) - 2025-06-08
### New Features 🎉
- fix: improve update logic for slides by @k1LoW in https://github.com/k1LoW/deck/pull/112
### Fix bug 🐛
- feat(deck): handle single-item bullet lists correctly by @k1LoW in https://github.com/k1LoW/deck/pull/111
### Other Changes
- chore(deps): bump google.golang.org/api from 0.234.0 to 0.235.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/108
- fix(deck): add integration test by @k1LoW in https://github.com/k1LoW/deck/pull/110

## [v0.19.1](https://github.com/k1LoW/deck/compare/v0.19.0...v0.19.1) - 2025-05-28
### Other Changes
- fix(md): handle allowed inline HTML elements by @k1LoW in https://github.com/k1LoW/deck/pull/107

## [v0.19.0](https://github.com/k1LoW/deck/compare/v0.18.0...v0.19.0) - 2025-05-28
### New Features 🎉
- feat(md): add support for class attributes in fragments by @k1LoW in https://github.com/k1LoW/deck/pull/102
- feat: support specifying styles for inline syntax in Markdown. by @k1LoW in https://github.com/k1LoW/deck/pull/104
- feat: add support for applying styles based on class names by @k1LoW in https://github.com/k1LoW/deck/pull/105
### Other Changes
- chore(deps): bump google.golang.org/api from 0.233.0 to 0.234.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/101

## [v0.18.0](https://github.com/k1LoW/deck/compare/v0.17.2...v0.18.0) - 2025-05-23
### New Features 🎉
- feat(oauth): implement PKCE for enhanced security in OAuth flow by @k1LoW in https://github.com/k1LoW/deck/pull/100
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/99

## [v0.17.2](https://github.com/k1LoW/deck/compare/v0.17.1...v0.17.2) - 2025-05-18
### Fix bug 🐛
- fix: empty link item causes panic by @k1LoW in https://github.com/k1LoW/deck/pull/97
### Other Changes
- test: add fuzzing test by @k1LoW in https://github.com/k1LoW/deck/pull/95

## [v0.17.1](https://github.com/k1LoW/deck/compare/v0.17.0...v0.17.1) - 2025-05-18
### Fix bug 🐛
- fix: empty list item causes panic by @k1LoW in https://github.com/k1LoW/deck/pull/94

## [v0.17.0](https://github.com/k1LoW/deck/compare/v0.16.3...v0.17.0) - 2025-05-16
### New Features 🎉
- feat(deck): add support for custom code span styles in slides by @k1LoW in https://github.com/k1LoW/deck/pull/91

## [v0.16.3](https://github.com/k1LoW/deck/compare/v0.16.2...v0.16.3) - 2025-05-14
### Other Changes
- feat(md): add support for inline code fragments ( parse only ) by @k1LoW in https://github.com/k1LoW/deck/pull/88
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/86

## [v0.16.2](https://github.com/k1LoW/deck/compare/v0.16.1...v0.16.2) - 2025-05-05
### Fix bug 🐛
- fix(deck): count string as UTF-16 by @k1LoW in https://github.com/k1LoW/deck/pull/84

## [v0.16.1](https://github.com/k1LoW/deck/compare/v0.16.0...v0.16.1) - 2025-05-05
### Fix bug 🐛
- feat(deck): add emoji-aware string counting ( does not support ligature ) by @k1LoW in https://github.com/k1LoW/deck/pull/83
### Other Changes
- chore(deps): bump google.golang.org/api from 0.230.0 to 0.231.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/81

## [v0.16.0](https://github.com/k1LoW/deck/compare/v0.15.0...v0.16.0) - 2025-04-29
### New Features 🎉
- feat: add verbose flag and improve log message handling by @k1LoW in https://github.com/k1LoW/deck/pull/77
### Other Changes
- fix: use context.Context by @k1LoW in https://github.com/k1LoW/deck/pull/79

## [v0.15.0](https://github.com/k1LoW/deck/compare/v0.14.0...v0.15.0) - 2025-04-28
### New Features 🎉
- feat: support `--watch` for watching update file. by @k1LoW in https://github.com/k1LoW/deck/pull/76

## [v0.14.0](https://github.com/k1LoW/deck/compare/v0.13.5...v0.14.0) - 2025-04-28
### Other Changes
- refactor: Introduce a new `deck.Slide` type by @k1LoW in https://github.com/k1LoW/deck/pull/73

## [v0.13.5](https://github.com/k1LoW/deck/compare/v0.13.4...v0.13.5) - 2025-04-28
### Fix bug 🐛
- fix(deck): support additional title placeholder type by @k1LoW in https://github.com/k1LoW/deck/pull/72
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/70

## [v0.13.4](https://github.com/k1LoW/deck/compare/v0.13.3...v0.13.4) - 2025-04-22
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/68

## [v0.13.3](https://github.com/k1LoW/deck/compare/v0.13.2...v0.13.3) - 2025-04-17
### Other Changes
- chore(deps): bump golang.org/x/net from 0.37.0 to 0.38.0 by @dependabot in https://github.com/k1LoW/deck/pull/66

## [v0.13.2](https://github.com/k1LoW/deck/compare/v0.13.1...v0.13.2) - 2025-04-08
### Other Changes
- chore(deps): bump google.golang.org/api from 0.227.0 to 0.228.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/64
- chore(deps): bump golang.org/x/oauth2 from 0.28.0 to 0.29.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/65

## [v0.13.1](https://github.com/k1LoW/deck/compare/v0.13.0...v0.13.1) - 2025-03-30

## [v0.13.0](https://github.com/k1LoW/deck/compare/v0.12.0...v0.13.0) - 2025-03-30
### Other Changes
- fix: apply default layout only when creating a new one by @k1LoW in https://github.com/k1LoW/deck/pull/61

## [v0.12.0](https://github.com/k1LoW/deck/compare/v0.11.3...v0.12.0) - 2025-03-27
### New Features 🎉
- fix: support using refresh expired tokens by @k1LoW in https://github.com/k1LoW/deck/pull/58
### Other Changes
- chore(deps): bump google.golang.org/api from 0.226.0 to 0.227.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/57

## [v0.11.3](https://github.com/k1LoW/deck/compare/v0.11.2...v0.11.3) - 2025-03-22
### Fix bug 🐛
- fix(deck): clear placeholder for speaker notes in applyPage function by @k1LoW in https://github.com/k1LoW/deck/pull/55

## [v0.11.2](https://github.com/k1LoW/deck/compare/v0.11.1...v0.11.2) - 2025-03-20

## [v0.11.1](https://github.com/k1LoW/deck/compare/v0.11.0...v0.11.1) - 2025-03-20
### Fix bug 🐛
- fix: handle missing auth code in getTokenFromWeb function by @k1LoW in https://github.com/k1LoW/deck/pull/52

## [v0.11.0](https://github.com/k1LoW/deck/compare/v0.10.4...v0.11.0) - 2025-03-20
### New Features 🎉
- feat: support applying both bold and italic by @k1LoW in https://github.com/k1LoW/deck/pull/50

## [v0.10.4](https://github.com/k1LoW/deck/compare/v0.10.3...v0.10.4) - 2025-03-20
### Fix bug 🐛
- fix(deck): fix newline handling by @k1LoW in https://github.com/k1LoW/deck/pull/48

## [v0.10.3](https://github.com/k1LoW/deck/compare/v0.10.2...v0.10.3) - 2025-03-20
### Fix bug 🐛
- fix(deck): fix newline handling by @k1LoW in https://github.com/k1LoW/deck/pull/46

## [v0.10.2](https://github.com/k1LoW/deck/compare/v0.10.1...v0.10.2) - 2025-03-19
### Fix bug 🐛
- fix: remove unnecessary newline by @k1LoW in https://github.com/k1LoW/deck/pull/43
- fix(deck): clear text style before deleting paragraph bullets by @k1LoW in https://github.com/k1LoW/deck/pull/45

## [v0.10.1](https://github.com/k1LoW/deck/compare/v0.10.0...v0.10.1) - 2025-03-19
### Other Changes
- fix(md): trim whitespace and newline characters from HTML Block fragment values by @k1LoW in https://github.com/k1LoW/deck/pull/41

## [v0.10.0](https://github.com/k1LoW/deck/compare/v0.9.1...v0.10.0) - 2025-03-19
### New Features 🎉
- feat(deck): change sig and add start/end option by @k1LoW in https://github.com/k1LoW/deck/pull/38
### Other Changes
- fix: add `--page` option instead of `--start/end` by @k1LoW in https://github.com/k1LoW/deck/pull/40

## [v0.9.1](https://github.com/k1LoW/deck/compare/v0.9.0...v0.9.1) - 2025-03-18
### Fix bug 🐛
- fix(deck): reset bullet indices per body by @k1LoW in https://github.com/k1LoW/deck/pull/37

## [v0.9.0](https://github.com/k1LoW/deck/compare/v0.8.1...v0.9.0) - 2025-03-17
### New Features 🎉
- feat(md): add support for converting `<br>` tags to newlines by @k1LoW in https://github.com/k1LoW/deck/pull/33
### Other Changes
- chore(deps): bump the dependencies group with 2 updates by @dependabot in https://github.com/k1LoW/deck/pull/35

## [v0.8.1](https://github.com/k1LoW/deck/compare/v0.8.0...v0.8.1) - 2025-03-16
### Fix bug 🐛
- fix(md): remove empty bodies from parsed pages by @k1LoW in https://github.com/k1LoW/deck/pull/32

## [v0.8.0](https://github.com/k1LoW/deck/compare/v0.7.1...v0.8.0) - 2025-03-16
### New Features 🎉
- feat: add freeze functionality to skip page modifications by @k1LoW in https://github.com/k1LoW/deck/pull/29

## [v0.7.1](https://github.com/k1LoW/deck/compare/v0.7.0...v0.7.1) - 2025-03-16
### Other Changes
- fix(deck): increase retry limits for HTTP client by @k1LoW in https://github.com/k1LoW/deck/pull/27

## [v0.7.0](https://github.com/k1LoW/deck/compare/v0.6.1...v0.7.0) - 2025-03-16
### New Features 🎉
- feat(logging): integrate slog for enhanced logging by @k1LoW in https://github.com/k1LoW/deck/pull/26

## [v0.6.1](https://github.com/k1LoW/deck/compare/v0.6.0...v0.6.1) - 2025-03-15
### Fix bug 🐛
- fix(deck): correct text style update logic by @k1LoW in https://github.com/k1LoW/deck/pull/24

## [v0.6.0](https://github.com/k1LoW/deck/compare/v0.5.0...v0.6.0) - 2025-03-15
### New Features 🎉
- feat: add support for italic text by @k1LoW in https://github.com/k1LoW/deck/pull/22

## [v0.5.0](https://github.com/k1LoW/deck/compare/v0.4.1...v0.5.0) - 2025-03-15
### New Features 🎉
- feat(deck): add support for setting speaker notes by @k1LoW in https://github.com/k1LoW/deck/pull/20

## [v0.4.1](https://github.com/k1LoW/deck/compare/v0.4.0...v0.4.1) - 2025-03-15
### Other Changes
- fix: print to STDOUT by @k1LoW in https://github.com/k1LoW/deck/pull/18

## [v0.4.0](https://github.com/k1LoW/deck/compare/v0.3.0...v0.4.0) - 2025-03-15
### New Features 🎉
- feat: support link by @k1LoW in https://github.com/k1LoW/deck/pull/16

## [v0.3.0](https://github.com/k1LoW/deck/compare/v0.2.0...v0.3.0) - 2025-03-15
### New Features 🎉
- feat(apply): add `--title` flag to update presentation title by @k1LoW in https://github.com/k1LoW/deck/pull/13
- feat(cmd): add `new` command to create presentations by @k1LoW in https://github.com/k1LoW/deck/pull/14

## [v0.2.0](https://github.com/k1LoW/deck/compare/v0.1.1...v0.2.0) - 2025-03-14
### New Features 🎉
- feat: open browser automatically for OAuth authentication by @k1LoW in https://github.com/k1LoW/deck/pull/10
### Fix bug 🐛
- fix: Use context instead of channel by @k1LoW in https://github.com/k1LoW/deck/pull/8
### Other Changes
- fix(md): handle leading horizontal line in markdown by @k1LoW in https://github.com/k1LoW/deck/pull/11

## [v0.1.1](https://github.com/k1LoW/deck/compare/v0.1.0...v0.1.1) - 2025-03-14

## [v0.1.0](https://github.com/k1LoW/deck/compare/v0.0.1...v0.1.0) - 2025-03-14
### New Features 🎉
- feat: add `ls` command to list Google Slides presentations by @k1LoW in https://github.com/k1LoW/deck/pull/3
- feat(deck): add retryable HTTP client with exponential backoff by @k1LoW in https://github.com/k1LoW/deck/pull/5
### Other Changes
- chore: change directories by @k1LoW in https://github.com/k1LoW/deck/pull/6

## [v0.0.1](https://github.com/k1LoW/deck/commits/v0.0.1) - 2025-03-14
### Other Changes
- chore(deps): bump google.golang.org/api from 0.225.0 to 0.226.0 in the dependencies group by @dependabot in https://github.com/k1LoW/deck/pull/1
