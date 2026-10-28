# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, render, inline CSS,
post-processing), measured on the reference development machine. Each budget is
the measured median plus ~60% headroom so normal noise does not trip it, while a
structural regression does.

Policy: CI prints the benchmark comparison but does **not** fail on it. Update this
table (and the numbers) whenever a change moves a budget materially, in the same
commit as that change.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians.
Fixtures that do not render yet (tracked by the parity baseline) are skipped.

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `button-complex.mjml` | 1,416,556 | 2,310,516 | 20,511 | 885,347 | 1,444,072 | 12,819 |
| `button.mjml` | 86,744 | 149,220 | 1,375 | 54,215 | 93,262 | 859 |
| `content.mjml` | 197,213 | 305,794 | 2,944 | 123,258 | 191,121 | 1,840 |
| `fluid-height-hero.mjml` | 109,904 | 186,447 | 1,704 | 68,690 | 116,529 | 1,065 |
| `head.mjml` | 128,293 | 195,893 | 1,972 | 80,183 | 122,433 | 1,232 |
| `image-complex.mjml` | 1,088,916 | 1,676,808 | 16,472 | 680,572 | 1,048,005 | 10,295 |
| `image.mjml` | 89,658 | 150,616 | 1,437 | 56,036 | 94,135 | 898 |
| `input_big.mjml` | 4,114,735 | 5,280,020 | 54,692 | 2,571,709 | 3,300,012 | 34,182 |
| `layout.mjml` | 353,658 | 563,738 | 5,536 | 221,036 | 352,336 | 3,460 |
| `simple-hero.mjml` | 110,087 | 183,444 | 1,701 | 68,804 | 114,652 | 1,063 |
| `simple.mjml` | 39,807 | 70,287 | 682 | 24,879 | 43,929 | 426 |
| `social.mjml` | 211,000 | 383,253 | 3,018 | 131,875 | 239,533 | 1,886 |
| `upstream/mjml--beautify-output--00.mjml` | 72,693 | 124,511 | 1,181 | 45,433 | 77,819 | 738 |
| `upstream/mjml--beautify-output--01.mjml` | 73,380 | 124,852 | 1,183 | 45,862 | 78,032 | 739 |
| `upstream/mjml--beautify-output--03.mjml` | 72,637 | 124,444 | 1,181 | 45,398 | 77,777 | 738 |
| `upstream/mjml--beautify-output--04.mjml` | 73,175 | 124,610 | 1,181 | 45,734 | 77,881 | 738 |
| `upstream/mjml--column-border-radius--00.mjml` | 81,932 | 136,308 | 1,319 | 51,207 | 85,192 | 824 |
| `upstream/mjml--html-comments--00.mjml` | 79,679 | 132,488 | 1,284 | 49,799 | 82,805 | 802 |
| `upstream/mjml--social-align--00.mjml` | 117,941 | 201,602 | 1,786 | 73,713 | 126,001 | 1,116 |
| `upstream/mjml--social-icon-height--00.mjml` | 126,336 | 225,117 | 1,869 | 78,960 | 140,698 | 1,168 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 97,920 | 155,055 | 1,576 | 61,200 | 96,909 | 985 |
| `upstream/mjml--wrapper-gap--00.mjml` | 214,295 | 359,524 | 3,367 | 133,934 | 224,702 | 2,104 |
| `upstream/parser-xml--preprocessors--00.mjml` | 73,552 | 124,906 | 1,191 | 45,970 | 78,066 | 744 |

Stage-level budgets are tracked from profiles for now; `BenchmarkParse` and
`BenchmarkParityComparison` isolate the parser and the test harness. Inline CSS
inlining is measured through the end-to-end benchmark until the inliner rewrite
(M3) exposes a separate seam.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
