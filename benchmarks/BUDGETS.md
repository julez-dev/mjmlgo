# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, render, inline CSS,
post-processing), measured on the reference development machine. Each budget is
the measured median plus ~60% headroom so normal noise does not trip it, while a
structural regression does.

Policy: CI prints the benchmark comparison but does **not** fail on it. Update this
table (and the numbers) whenever a change moves a budget materially, in the same
commit as that change.

Measured with Go 1.27 on an AMD Ryzen 9 9950X3D (GOMAXPROCS 32),
`-benchmem -count=3`, medians. Fixtures that do not render yet (tracked by the
parity baseline) are skipped.

Last full re-measurement: the font-usage detection rewrite (`component/fonts.go`),
which removed per-render regexp compilation and moved every row by 2-5x.

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `button-complex.mjml` | 1,655,732 | 2,911,008 | 21,200 | 1,034,832 | 1,819,380 | 13,250 |
| `button.mjml` | 102,532 | 200,271 | 1,471 | 64,082 | 125,169 | 919 |
| `content.mjml` | 252,383 | 398,301 | 3,168 | 157,739 | 248,938 | 1,980 |
| `fluid-height-hero.mjml` | 141,066 | 265,295 | 1,860 | 88,166 | 165,809 | 1,162 |
| `head.mjml` | 153,828 | 260,114 | 2,130 | 96,142 | 162,571 | 1,331 |
| `image-complex.mjml` | 1,296,565 | 2,087,756 | 17,031 | 810,353 | 1,304,847 | 10,644 |
| `image.mjml` | 108,464 | 201,320 | 1,533 | 67,790 | 125,825 | 958 |
| `input_big.mjml` | 4,802,720 | 6,420,439 | 55,898 | 3,001,700 | 4,012,774 | 34,936 |
| `layout.mjml` | 423,085 | 768,511 | 5,986 | 264,428 | 480,319 | 3,741 |
| `simple-hero.mjml` | 134,524 | 261,416 | 1,856 | 84,077 | 163,385 | 1,160 |
| `simple.mjml` | 46,588 | 96,712 | 725 | 29,117 | 60,445 | 453 |
| `social.mjml` | 247,986 | 480,018 | 3,231 | 154,991 | 300,011 | 2,019 |
| `upstream/mjml--beautify-output--00.mjml` | 90,804 | 178,234 | 1,368 | 56,752 | 111,396 | 855 |
| `upstream/mjml--beautify-output--01.mjml` | 91,458 | 178,402 | 1,368 | 57,161 | 111,501 | 855 |
| `upstream/mjml--beautify-output--03.mjml` | 91,866 | 178,162 | 1,368 | 57,416 | 111,351 | 855 |
| `upstream/mjml--beautify-output--04.mjml` | 91,426 | 178,432 | 1,368 | 57,141 | 111,520 | 855 |
| `upstream/mjml--column-border-radius--00.mjml` | 105,789 | 197,316 | 1,504 | 66,118 | 123,322 | 940 |
| `upstream/mjml--html-comments--00.mjml` | 99,844 | 193,362 | 1,469 | 62,402 | 120,851 | 918 |
| `upstream/mjml--social-align--00.mjml` | 144,466 | 282,618 | 1,983 | 90,291 | 176,636 | 1,239 |
| `upstream/mjml--social-icon-height--00.mjml` | 151,676 | 295,370 | 2,063 | 94,797 | 184,606 | 1,289 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 112,775 | 209,772 | 1,680 | 70,484 | 131,107 | 1,050 |
| `upstream/mjml--wrapper-gap--00.mjml` | 252,383 | 473,373 | 3,604 | 157,739 | 295,858 | 2,252 |
| `upstream/parser-xml--preprocessors--00.mjml` | 92,199 | 178,738 | 1,376 | 57,624 | 111,711 | 860 |

Stage-level budgets are tracked from profiles for now; `BenchmarkParse` and
`BenchmarkParityComparison` isolate the parser and the test harness. Inline CSS
inlining is measured through the end-to-end benchmark until the inliner rewrite
(M3) exposes a separate seam. `BenchmarkUsedFonts` guards the font-usage scan on
a 24 KB rendered body (~13 µs, 2 allocs); it used to be the largest single cost
in the pipeline because it compiled two regexps per registered font per render.

## Reproducing

```bash
# full pipeline, per fixture
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
# font usage scan over a 24 KB body
go test ./component -run XXX -bench BenchmarkUsedFonts -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
