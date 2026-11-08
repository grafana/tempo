## 0.2.1
 - Fixed handling of variable-width clock elements (`3`, `4`, `5`) so layouts stay in sync when hours, minutes, or seconds use one or two digits ([#15](https://github.com/elastic/lunes/issues/15)).
 - Failed reads of `_2`, `_2006`, and `__2` now returns `ErrLayoutMismatch` errors.
 - Fixed typo in the `ErrUnsupportedLayoutElem` message (`is not support by` to `is not supported by`).

## 0.2.0
 - Updated Unicode CLDR locale data to v48 for broader language coverage and corrected region aliases.
 - Added lazy loading of locale tables, reducing initial memory footprint and improving startup performance.
