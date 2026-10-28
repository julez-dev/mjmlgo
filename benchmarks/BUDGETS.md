# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, validate, render, inline
CSS, post-processing), measured on the reference development machine. Each budget
is the measured median plus ~60% headroom so normal noise does not trip it while a
structural regression does.

Policy: CI prints benchmark results but does not fail on them. Update this table
in the same commit as a change that moves a budget materially.

Measured with Go 1.24 on an AMD Ryzen 9 9950X3D, `-benchmem -count=3`, medians,
over the whole parity corpus. A fixture with a `.cli-args` sidecar is measured
with the options it records, from memory; fixtures with `mj-include` are measured
through `Renderer.RenderString` (fs reads included).

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `accordion.mjml` | 115,315 | 195,936 | 1,504 | 72,072 | 122,460 | 940 |
| `button-complex.mjml` | 890,019 | 1,346,067 | 12,658 | 556,262 | 841,292 | 7,911 |
| `button.mjml` | 55,122 | 83,952 | 890 | 34,451 | 52,470 | 556 |
| `carousel-thumbnails-hidden.mjml` | 83,202 | 209,382 | 1,429 | 52,001 | 130,864 | 893 |
| `carousel.mjml` | 115,430 | 306,107 | 1,962 | 72,144 | 191,317 | 1,226 |
| `comments-between-components.mjml` | 111,466 | 181,949 | 1,806 | 69,666 | 113,718 | 1,129 |
| `comments-containers.mjml` | 195,051 | 303,357 | 2,216 | 121,907 | 189,598 | 1,385 |
| `comments-head.mjml` | 64,186 | 81,635 | 917 | 40,116 | 51,022 | 573 |
| `comments-hero.mjml` | 108,141 | 185,584 | 1,810 | 67,588 | 115,990 | 1,131 |
| `comments-hero-inline.mjml` | 167,074 | 284,554 | 2,266 | 104,421 | 177,846 | 1,416 |
| `comments-keep-false.mjml` | 58,197 | 86,696 | 907 | 36,373 | 54,185 | 567 |
| `comments-raw.mjml` | 55,634 | 89,237 | 880 | 34,771 | 55,773 | 550 |
| `component-head-styles.mjml` | 107,646 | 195,893 | 1,582 | 67,279 | 122,433 | 989 |
| `content.mjml` | 127,392 | 179,747 | 1,885 | 79,620 | 112,342 | 1,178 |
| `fluid-height-hero.mjml` | 71,202 | 121,117 | 1,190 | 44,501 | 75,698 | 744 |
| `head-attributes-class.mjml` | 101,235 | 145,720 | 1,538 | 63,272 | 91,075 | 961 |
| `head-attributes-per-tag.mjml` | 76,782 | 117,934 | 1,163 | 47,989 | 73,709 | 727 |
| `head-fonts-custom.mjml` | 70,798 | 107,158 | 1,133 | 44,249 | 66,974 | 708 |
| `head-fonts-default.mjml` | 56,667 | 82,904 | 979 | 35,417 | 51,815 | 612 |
| `head-html-attributes.mjml` | 143,320 | 188,739 | 2,008 | 89,575 | 117,962 | 1,255 |
| `head-owa.mjml` | 51,854 | 74,283 | 858 | 32,409 | 46,427 | 536 |
| `head.mjml` | 140,418 | 182,531 | 1,811 | 87,761 | 114,082 | 1,132 |
| `ignore-includes/main.mjml` | 56,446 | 81,896 | 909 | 35,279 | 51,185 | 568 |
| `image-complex.mjml` | 692,723 | 973,618 | 10,530 | 432,952 | 608,511 | 6,581 |
| `image.mjml` | 56,501 | 87,266 | 973 | 35,313 | 54,541 | 608 |
| `includes/main.mjml` | 415,966 | 575,293 | 5,032 | 259,979 | 359,558 | 3,145 |
| `input_big.mjml` | 4,005,432 | 4,834,640 | 41,744 | 2,503,395 | 3,021,650 | 26,090 |
| `layout.mjml` | 228,400 | 352,352 | 3,758 | 142,750 | 220,220 | 2,349 |
| `navbar.mjml` | 79,104 | 129,451 | 1,182 | 49,440 | 80,907 | 739 |
| `printer-support.mjml` | 74,275 | 112,021 | 1,229 | 46,422 | 70,013 | 768 |
| `raw-in-column.mjml` | 55,442 | 89,648 | 880 | 34,651 | 56,030 | 550 |
| `raw-in-column-inline.mjml` | 123,653 | 216,232 | 1,707 | 77,283 | 135,145 | 1,067 |
| `raw-in-group.mjml` | 82,048 | 115,821 | 1,381 | 51,280 | 72,388 | 863 |
| `raw-in-section.mjml` | 69,170 | 101,514 | 1,168 | 43,231 | 63,446 | 730 |
| `raw-position-file-start.mjml` | 48,400 | 77,832 | 848 | 30,250 | 48,645 | 530 |
| `raw-rows-in-column-inline.mjml` | 109,909 | 189,722 | 1,494 | 68,693 | 118,576 | 934 |
| `raw-table-in-column-inline.mjml` | 107,902 | 188,717 | 1,488 | 67,439 | 117,948 | 930 |
| `raw.mjml` | 52,994 | 89,218 | 878 | 33,121 | 55,761 | 549 |
| `section-container-width.mjml` | 88,325 | 128,483 | 1,525 | 55,203 | 80,302 | 953 |
| `simple-hero.mjml` | 72,030 | 119,402 | 1,195 | 45,019 | 74,626 | 747 |
| `simple.mjml` | 22,986 | 36,661 | 430 | 14,366 | 22,913 | 269 |
| `social.mjml` | 132,069 | 258,115 | 2,019 | 82,543 | 161,322 | 1,262 |
| `table-width-auto.mjml` | 51,290 | 77,824 | 864 | 32,056 | 48,640 | 540 |
| `upstream/mjml--accordion-fontFamily--00.mjml` | 116,856 | 206,603 | 1,682 | 73,035 | 129,127 | 1,051 |
| `upstream/mjml--accordion-padding--00.mjml` | 79,565 | 130,443 | 1,082 | 49,728 | 81,527 | 676 |
| `upstream/mjml--accordionTitle-fontWeight--00.mjml` | 132,187 | 196,350 | 1,493 | 82,617 | 122,719 | 933 |
| `upstream/mjml--beautify-output--00.mjml` | 46,667 | 71,278 | 821 | 29,167 | 44,549 | 513 |
| `upstream/mjml--beautify-output--01.mjml` | 48,626 | 71,450 | 822 | 30,391 | 44,656 | 514 |
| `upstream/mjml--beautify-output--02.mjml` | 44,197 | 70,715 | 722 | 27,623 | 44,197 | 451 |
| `upstream/mjml--beautify-output--03.mjml` | 46,634 | 71,187 | 821 | 29,146 | 44,492 | 513 |
| `upstream/mjml--beautify-output--04.mjml` | 47,040 | 71,509 | 822 | 29,400 | 44,693 | 514 |
| `upstream/mjml--carousel-hoverSupported--00.mjml` | 113,571 | 302,114 | 1,866 | 70,982 | 188,821 | 1,166 |
| `upstream/mjml--column-border-radius--00.mjml` | 55,304 | 87,846 | 946 | 34,565 | 54,904 | 591 |
| `upstream/mjml--html-attributes--00.mjml` | 164,789 | 263,994 | 2,373 | 102,993 | 164,996 | 1,483 |
| `upstream/mjml--html-comments--00.mjml` | 49,789 | 75,518 | 826 | 31,118 | 47,199 | 516 |
| `upstream/mjml--htmlmin-ignore--00.mjml` | 44,328 | 71,264 | 722 | 27,705 | 44,540 | 451 |
| `upstream/mjml--htmlmin-ignore--01.mjml` | 45,893 | 73,168 | 723 | 28,683 | 45,730 | 452 |
| `upstream/mjml--htmlmin-ignore--02.mjml` | 45,851 | 70,669 | 722 | 28,657 | 44,168 | 451 |
| `upstream/mjml--navbar-ico-padding--00.mjml` | 76,022 | 130,907 | 1,154 | 47,514 | 81,817 | 721 |
| `upstream/mjml--social-align--00.mjml` | 77,904 | 133,446 | 1,213 | 48,690 | 83,404 | 758 |
| `upstream/mjml--social-icon-height--00.mjml` | 82,794 | 143,960 | 1,270 | 51,746 | 89,975 | 794 |
| `upstream/mjml--table-cellspacing--00.mjml` | 60,485 | 89,805 | 890 | 37,803 | 56,128 | 556 |
| `upstream/mjml--tableWidth--00.mjml` | 184,218 | 274,283 | 2,634 | 115,136 | 171,427 | 1,646 |
| `upstream/mjml--template-syntax-sanitization--00.mjml` | 44,066 | 70,858 | 722 | 27,541 | 44,286 | 451 |
| `upstream/mjml--template-syntax-sanitization--01.mjml` | 50,995 | 74,790 | 856 | 31,872 | 46,744 | 535 |
| `upstream/mjml--template-syntax-sanitization--02.mjml` | 43,710 | 69,941 | 720 | 27,319 | 43,713 | 450 |
| `upstream/mjml--template-syntax-sanitization--03.mjml` | 44,986 | 69,867 | 720 | 28,116 | 43,667 | 450 |
| `upstream/mjml--template-syntax-sanitization--04.mjml` | 47,123 | 72,256 | 762 | 29,452 | 45,160 | 476 |
| `upstream/mjml--template-syntax-sanitization--05.mjml` | 57,253 | 69,851 | 720 | 35,783 | 43,657 | 450 |
| `upstream/mjml--template-syntax-sanitization--06.mjml` | 44,600 | 70,136 | 720 | 27,875 | 43,835 | 450 |
| `upstream/mjml--template-syntax-sanitization--07.mjml` | 44,522 | 70,080 | 720 | 27,826 | 43,800 | 450 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 58,582 | 85,499 | 1,006 | 36,614 | 53,437 | 629 |
| `upstream/mjml--wrapper-gap--00.mjml` | 129,186 | 203,210 | 2,168 | 80,741 | 127,006 | 1,355 |
| `upstream/parser-xml--preprocessors--00.mjml` | 51,462 | 74,498 | 886 | 32,164 | 46,561 | 554 |

## Inline stage profile

The inline stage no longer round-trips the document through `html.Parse` and
`html.Render`. It tokenizes the markup the components emitted, matches the
selectors against that tree and rewrites only the start tags a rule changed
(`BenchmarkInlineStage`, document already rendered, medians of 7):

| Fixture | Before ns/op | After ns/op | Before allocs/op | After allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `input_big.mjml` | 817,000 | 610,000 | 7,815 | 6,830 |
| `head.mjml` | 25,400 | 19,100 | 298 | 255 |

End to end that is `input_big.mjml` 2.59 ms -> 2.31 ms (35,781 -> 34,776 allocs/op)
and `head.mjml` 94.6 us -> 83.7 us (1,493 -> 1,448 allocs/op), measured by running
the pre-change and the post-change tree alternately in one session; the per-fixture
numbers above come from single runs of each tree and are medians of 7 and 5.

What the stage costs now, from its CPU profile: about half is tokenizing the
emitted markup and building the tree, a third is selector matching (one full tree
traversal per rule) and the rest is the writer that patches the changed start
tags. Removing the tokenizer would mean components emitting structured elements
instead of text, which is a much larger change and is not planned.

## Known follow-ups

- Selector matching is one full tree traversal per rule, because that is the shape
  of the selector engine's API. Indexing rules by their rightmost compound
  selector is the next win in the inline stage; the profile above sizes it at
  roughly a third of the stage.
- The numbers in the inline-stage table below were measured before the validation
  work landed and are re-measured on the merged tree.

`BenchmarkParse` and `BenchmarkParityComparison` isolate the parser and the test
harness.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=5
go test ./ -run XXX -bench BenchmarkInlineStage -benchmem -count=5
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
