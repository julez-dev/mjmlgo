# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, render, inline CSS,
post-processing), measured on the reference development machine. Each budget is
the measured median plus ~60% headroom so normal noise does not trip it while a
structural regression does.

Policy: CI prints benchmark results but does not fail on them. Update this table
in the same commit as a change that moves a budget materially.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians,
over the whole parity corpus. Fixtures with `mj-include` are measured through
`Renderer.RenderString` (fs reads included).

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `accordion.mjml` | 129,384 | 232,765 | 2,036 | 80,865 | 145,478 | 1,272 |
| `button-complex.mjml` | 1,064,328 | 1,685,015 | 17,204 | 665,205 | 1,053,134 | 10,752 |
| `button.mjml` | 59,644 | 100,039 | 1,093 | 37,277 | 62,524 | 683 |
| `carousel-thumbnails-hidden.mjml` | 90,736 | 227,237 | 1,604 | 56,710 | 142,023 | 1,002 |
| `carousel.mjml` | 124,533 | 326,957 | 2,146 | 77,833 | 204,348 | 1,341 |
| `component-head-styles.mjml` | 125,818 | 229,548 | 2,048 | 78,636 | 143,467 | 1,280 |
| `content.mjml` | 141,605 | 211,836 | 2,335 | 88,503 | 132,397 | 1,459 |
| `fluid-height-hero.mjml` | 77,645 | 138,290 | 1,384 | 48,528 | 86,431 | 865 |
| `head-attributes-class.mjml` | 109,799 | 175,565 | 1,935 | 68,624 | 109,728 | 1,209 |
| `head-attributes-per-tag.mjml` | 85,920 | 132,205 | 1,487 | 53,700 | 82,628 | 929 |
| `head-fonts-default.mjml` | 63,605 | 100,272 | 1,212 | 39,753 | 62,670 | 757 |
| `head-html-attributes.mjml` | 146,866 | 206,802 | 2,244 | 91,791 | 129,251 | 1,402 |
| `head-owa.mjml` | 53,568 | 87,565 | 1,032 | 33,480 | 54,728 | 645 |
| `head.mjml` | 141,820 | 199,893 | 2,039 | 88,637 | 124,933 | 1,274 |
| `image-complex.mjml` | 811,037 | 1,191,317 | 13,861 | 506,898 | 744,573 | 8,663 |
| `image.mjml` | 60,530 | 100,328 | 1,151 | 37,831 | 62,705 | 719 |
| `includes/main.mjml` | 444,533 | 612,285 | 5,599 | 277,833 | 382,678 | 3,499 |
| `input_big.mjml` | 3,997,770 | 4,894,837 | 48,576 | 2,498,606 | 3,059,273 | 30,360 |
| `layout.mjml` | 264,048 | 427,832 | 4,772 | 165,030 | 267,395 | 2,982 |
| `navbar.mjml` | 87,735 | 151,888 | 1,485 | 54,834 | 94,930 | 928 |
| `raw-in-column.mjml` | 59,221 | 104,116 | 1,068 | 37,013 | 65,072 | 667 |
| `raw-in-group.mjml` | 87,320 | 139,301 | 1,666 | 54,575 | 87,063 | 1,041 |
| `raw-in-section.mjml` | 76,072 | 123,368 | 1,453 | 47,545 | 77,105 | 908 |
| `raw-position-file-start.mjml` | 53,015 | 91,047 | 1,021 | 33,134 | 56,904 | 638 |
| `raw.mjml` | 58,327 | 103,730 | 1,066 | 36,454 | 64,831 | 666 |
| `section-container-width.mjml` | 105,018 | 151,936 | 1,837 | 65,636 | 94,960 | 1,148 |
| `simple-hero.mjml` | 188,730 | 135,552 | 1,391 | 117,956 | 84,720 | 869 |
| `simple.mjml` | 28,436 | 41,988 | 506 | 17,772 | 26,242 | 316 |
| `social.mjml` | 167,908 | 295,701 | 2,474 | 104,942 | 184,813 | 1,546 |
| `table-width-auto.mjml` | 72,280 | 90,786 | 1,042 | 45,175 | 56,741 | 651 |
| `upstream/mjml--accordion-fontFamily--00.mjml` | 135,952 | 246,076 | 2,250 | 84,970 | 153,797 | 1,406 |
| `upstream/mjml--accordion-padding--00.mjml` | 87,061 | 153,154 | 1,397 | 54,413 | 95,721 | 873 |
| `upstream/mjml--accordionTitle-fontWeight--00.mjml` | 138,815 | 223,976 | 1,898 | 86,759 | 139,985 | 1,186 |
| `upstream/mjml--beautify-output--00.mjml` | 53,080 | 84,316 | 996 | 33,175 | 52,697 | 622 |
| `upstream/mjml--beautify-output--01.mjml` | 56,252 | 84,474 | 997 | 35,157 | 52,796 | 623 |
| `upstream/mjml--beautify-output--02.mjml` | 46,346 | 79,938 | 844 | 28,966 | 49,961 | 527 |
| `upstream/mjml--beautify-output--03.mjml` | 51,442 | 84,311 | 996 | 32,151 | 52,694 | 622 |
| `upstream/mjml--beautify-output--04.mjml` | 51,850 | 84,751 | 997 | 32,406 | 52,969 | 623 |
| `upstream/mjml--carousel-hoverSupported--00.mjml` | 123,224 | 323,124 | 2,052 | 77,015 | 201,952 | 1,282 |
| `upstream/mjml--column-border-radius--00.mjml` | 65,212 | 101,650 | 1,130 | 40,757 | 63,531 | 706 |
| `upstream/mjml--html-attributes--00.mjml` | 172,229 | 250,223 | 2,466 | 107,643 | 156,389 | 1,541 |
| `upstream/mjml--html-comments--00.mjml` | 54,540 | 88,583 | 1,000 | 34,087 | 55,364 | 625 |
| `upstream/mjml--htmlmin-ignore--00.mjml` | 48,272 | 87,007 | 845 | 30,170 | 54,379 | 528 |
| `upstream/mjml--htmlmin-ignore--01.mjml` | 47,720 | 82,536 | 845 | 29,825 | 51,585 | 528 |
| `upstream/mjml--htmlmin-ignore--02.mjml` | 45,996 | 80,098 | 844 | 28,747 | 50,061 | 527 |
| `upstream/mjml--navbar-ico-padding--00.mjml` | 87,690 | 153,170 | 1,460 | 54,806 | 95,731 | 912 |
| `upstream/mjml--social-align--00.mjml` | 87,063 | 155,628 | 1,500 | 54,414 | 97,267 | 937 |
| `upstream/mjml--social-icon-height--00.mjml` | 91,594 | 166,677 | 1,559 | 57,246 | 104,173 | 974 |
| `upstream/mjml--table-cellspacing--00.mjml` | 63,490 | 104,252 | 1,071 | 39,681 | 65,157 | 669 |
| `upstream/mjml--tableWidth--00.mjml` | 207,588 | 326,264 | 3,388 | 129,742 | 203,915 | 2,117 |
| `upstream/mjml--template-syntax-sanitization--00.mjml` | 46,671 | 80,117 | 844 | 29,169 | 50,073 | 527 |
| `upstream/mjml--template-syntax-sanitization--01.mjml` | 53,922 | 88,074 | 1,031 | 33,701 | 55,046 | 644 |
| `upstream/mjml--template-syntax-sanitization--02.mjml` | 45,308 | 79,328 | 842 | 28,317 | 49,580 | 526 |
| `upstream/mjml--template-syntax-sanitization--03.mjml` | 45,455 | 79,288 | 842 | 28,409 | 49,555 | 526 |
| `upstream/mjml--template-syntax-sanitization--04.mjml` | 48,396 | 81,360 | 884 | 30,247 | 50,850 | 552 |
| `upstream/mjml--template-syntax-sanitization--05.mjml` | 45,352 | 79,319 | 842 | 28,345 | 49,574 | 526 |
| `upstream/mjml--template-syntax-sanitization--06.mjml` | 45,700 | 79,444 | 842 | 28,562 | 49,652 | 526 |
| `upstream/mjml--template-syntax-sanitization--07.mjml` | 45,512 | 79,384 | 842 | 28,445 | 49,615 | 526 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 66,101 | 102,514 | 1,242 | 41,313 | 64,071 | 776 |
| `upstream/mjml--wrapper-gap--00.mjml` | 144,255 | 245,420 | 2,736 | 90,159 | 153,387 | 1,710 |
| `upstream/parser-xml--preprocessors--00.mjml` | 51,828 | 84,706 | 1,004 | 32,392 | 52,941 | 627 |

`BenchmarkParse` and `BenchmarkParityComparison` isolate the parser and the test
harness. The remaining inline-stage cost is dominated by the post-render
`html.Parse`/`html.Render` round-trip (measured at ~0.54 ms of `input_big`, ~65%
of the inline stage); removing it is a tracked follow-up.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=3
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
