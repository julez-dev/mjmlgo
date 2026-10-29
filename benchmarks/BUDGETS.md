# Performance budgets

Budgets for `BenchmarkRenderMJML` (full pipeline: parse, validate, render, inline
CSS, post-processing), measured on the reference development machine. Each budget
is the measured median plus ~60% headroom so normal noise does not trip it while a
structural regression does.

Policy: CI prints benchmark results but does not fail on them. Update this table
in the same commit as a change that moves a budget materially.

Measured with Go 1.27 on an AMD Ryzen 9 9950X3D, `-benchmem -count=5`, medians,
over the whole parity corpus, in one run on the tree this table was committed
with. A fixture with a `.cli-args` sidecar is measured with the options it
records, from memory; fixtures with `mj-include` are measured through
`Renderer.RenderString` (fs reads included).

| Fixture | Budget ns/op | Budget B/op | Budget allocs/op | Measured ns/op | Measured B/op | Measured allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `accordion.mjml` | 114,269 | 199,720 | 1,470 | 71,418 | 124,825 | 919 |
| `button-complex.mjml` | 836,429 | 1,346,358 | 11,910 | 522,768 | 841,474 | 7,444 |
| `button.mjml` | 49,736 | 84,461 | 846 | 31,085 | 52,788 | 529 |
| `carousel-thumbnails-hidden.mjml` | 85,909 | 211,445 | 1,347 | 53,693 | 132,153 | 842 |
| `carousel.mjml` | 115,918 | 309,693 | 1,861 | 72,449 | 193,558 | 1,163 |
| `comments-between-components.mjml` | 102,019 | 182,747 | 1,710 | 63,762 | 114,217 | 1,069 |
| `comments-containers.mjml` | 154,779 | 306,854 | 2,120 | 96,737 | 191,784 | 1,325 |
| `comments-head.mjml` | 50,752 | 82,314 | 882 | 31,720 | 51,446 | 551 |
| `comments-hero-inline.mjml` | 134,440 | 226,622 | 2,018 | 84,025 | 141,639 | 1,261 |
| `comments-hero.mjml` | 110,307 | 184,790 | 1,704 | 68,942 | 115,494 | 1,065 |
| `comments-keep-false.mjml` | 52,605 | 87,104 | 869 | 32,878 | 54,440 | 543 |
| `comments-raw.mjml` | 49,698 | 89,698 | 843 | 31,061 | 56,061 | 527 |
| `component-head-styles.mjml` | 102,638 | 198,067 | 1,518 | 64,149 | 123,792 | 949 |
| `content.mjml` | 118,728 | 178,955 | 1,792 | 74,205 | 111,847 | 1,120 |
| `fluid-height-hero.mjml` | 67,229 | 121,446 | 1,122 | 42,018 | 75,904 | 701 |
| `head-attributes-class.mjml` | 92,158 | 147,411 | 1,474 | 57,599 | 92,132 | 921 |
| `head-attributes-per-tag.mjml` | 70,853 | 118,682 | 1,114 | 44,283 | 74,176 | 696 |
| `head-fonts-custom.mjml` | 66,381 | 107,962 | 1,082 | 41,488 | 67,476 | 676 |
| `head-fonts-default.mjml` | 51,890 | 83,072 | 936 | 32,431 | 51,920 | 585 |
| `head-html-attributes.mjml` | 122,234 | 188,082 | 1,882 | 76,396 | 117,551 | 1,176 |
| `head-owa.mjml` | 45,565 | 74,515 | 821 | 28,478 | 46,572 | 513 |
| `head.mjml` | 119,246 | 183,853 | 1,699 | 74,529 | 114,908 | 1,062 |
| `ignore-includes/main.mjml` | 50,771 | 82,304 | 872 | 31,732 | 51,440 | 545 |
| `image-complex.mjml` | 644,445 | 964,478 | 9,970 | 402,778 | 602,799 | 6,231 |
| `image.mjml` | 51,930 | 87,384 | 930 | 32,456 | 54,615 | 581 |
| `includes/main.mjml` | 354,768 | 534,315 | 4,522 | 221,730 | 333,947 | 2,826 |
| `inline-cascade.mjml` | 265,102 | 317,109 | 4,120 | 165,689 | 198,193 | 2,575 |
| `inline-implied-end-tags.mjml` | 189,365 | 250,520 | 3,070 | 118,353 | 156,575 | 1,919 |
| `inline-self-closed-html.mjml` | 131,438 | 179,685 | 2,155 | 82,149 | 112,303 | 1,347 |
| `inline-stray-void-end-tag.mjml` | 95,661 | 143,013 | 1,467 | 59,788 | 89,383 | 917 |
| `input_big.mjml` | 3,393,758 | 4,870,384 | 38,107 | 2,121,099 | 3,043,990 | 23,817 |
| `layout.mjml` | 222,805 | 351,661 | 3,520 | 139,253 | 219,788 | 2,200 |
| `navbar.mjml` | 75,512 | 130,347 | 1,133 | 47,195 | 81,467 | 708 |
| `printer-support.mjml` | 70,818 | 112,096 | 1,163 | 44,261 | 70,060 | 727 |
| `raw-in-column-inline.mjml` | 97,442 | 156,717 | 1,450 | 60,901 | 97,948 | 906 |
| `raw-in-column.mjml` | 50,595 | 89,909 | 843 | 31,622 | 56,193 | 527 |
| `raw-in-group.mjml` | 75,642 | 116,922 | 1,310 | 47,276 | 73,076 | 819 |
| `raw-in-section.mjml` | 63,899 | 101,259 | 1,104 | 39,937 | 63,287 | 690 |
| `raw-position-file-start.mjml` | 45,410 | 77,990 | 810 | 28,381 | 48,744 | 506 |
| `raw-rows-in-column-inline.mjml` | 91,752 | 151,387 | 1,349 | 57,345 | 94,617 | 843 |
| `raw-table-in-column-inline.mjml` | 90,554 | 149,296 | 1,325 | 56,596 | 93,310 | 828 |
| `raw.mjml` | 50,424 | 89,442 | 842 | 31,515 | 55,901 | 526 |
| `section-container-width.mjml` | 82,982 | 128,062 | 1,440 | 51,864 | 80,039 | 900 |
| `simple-hero.mjml` | 68,466 | 119,578 | 1,130 | 42,791 | 74,736 | 706 |
| `simple.mjml` | 20,946 | 36,568 | 410 | 13,091 | 22,855 | 256 |
| `social.mjml` | 129,086 | 260,011 | 1,917 | 80,679 | 162,507 | 1,198 |
| `table-width-auto.mjml` | 46,760 | 77,858 | 826 | 29,225 | 48,661 | 516 |
| `upstream/mjml--accordion-fontFamily--00.mjml` | 113,174 | 208,789 | 1,618 | 70,734 | 130,493 | 1,011 |
| `upstream/mjml--accordion-padding--00.mjml` | 72,555 | 132,731 | 1,048 | 45,347 | 82,957 | 655 |
| `upstream/mjml--accordionTitle-fontWeight--00.mjml` | 121,082 | 198,539 | 1,458 | 75,676 | 124,087 | 911 |
| `upstream/mjml--beautify-output--00.mjml` | 43,267 | 71,251 | 784 | 27,042 | 44,532 | 490 |
| `upstream/mjml--beautify-output--01.mjml` | 43,661 | 71,434 | 786 | 27,288 | 44,646 | 491 |
| `upstream/mjml--beautify-output--02.mjml` | 40,667 | 70,718 | 690 | 25,417 | 44,199 | 431 |
| `upstream/mjml--beautify-output--03.mjml` | 43,110 | 71,142 | 784 | 26,944 | 44,464 | 490 |
| `upstream/mjml--beautify-output--04.mjml` | 44,034 | 71,456 | 786 | 27,521 | 44,660 | 491 |
| `upstream/mjml--carousel-hoverSupported--00.mjml` | 115,339 | 305,587 | 1,770 | 72,087 | 190,992 | 1,106 |
| `upstream/mjml--column-border-radius--00.mjml` | 52,389 | 87,922 | 904 | 32,743 | 54,951 | 565 |
| `upstream/mjml--html-attributes--00.mjml` | 132,747 | 214,944 | 1,997 | 82,967 | 134,340 | 1,248 |
| `upstream/mjml--html-comments--00.mjml` | 46,074 | 75,522 | 789 | 28,796 | 47,201 | 493 |
| `upstream/mjml--htmlmin-ignore--00.mjml` | 41,334 | 71,213 | 690 | 25,834 | 44,508 | 431 |
| `upstream/mjml--htmlmin-ignore--01.mjml` | 42,640 | 73,301 | 691 | 26,650 | 45,813 | 432 |
| `upstream/mjml--htmlmin-ignore--02.mjml` | 41,042 | 70,797 | 690 | 25,651 | 44,248 | 431 |
| `upstream/mjml--navbar-ico-padding--00.mjml` | 72,493 | 131,403 | 1,104 | 45,308 | 82,127 | 690 |
| `upstream/mjml--social-align--00.mjml` | 72,893 | 134,118 | 1,157 | 45,558 | 83,824 | 723 |
| `upstream/mjml--social-icon-height--00.mjml` | 77,877 | 144,733 | 1,210 | 48,673 | 90,458 | 756 |
| `upstream/mjml--table-cellspacing--00.mjml` | 55,235 | 90,893 | 854 | 34,522 | 56,808 | 534 |
| `upstream/mjml--tableWidth--00.mjml` | 170,379 | 272,368 | 2,491 | 106,487 | 170,230 | 1,557 |
| `upstream/mjml--template-syntax-sanitization--00.mjml` | 40,949 | 70,789 | 690 | 25,593 | 44,243 | 431 |
| `upstream/mjml--template-syntax-sanitization--01.mjml` | 45,414 | 74,856 | 819 | 28,384 | 46,785 | 512 |
| `upstream/mjml--template-syntax-sanitization--02.mjml` | 40,176 | 69,907 | 688 | 25,110 | 43,692 | 430 |
| `upstream/mjml--template-syntax-sanitization--03.mjml` | 40,358 | 69,918 | 688 | 25,224 | 43,699 | 430 |
| `upstream/mjml--template-syntax-sanitization--04.mjml` | 43,264 | 72,269 | 730 | 27,040 | 45,168 | 456 |
| `upstream/mjml--template-syntax-sanitization--05.mjml` | 40,368 | 69,938 | 688 | 25,230 | 43,711 | 430 |
| `upstream/mjml--template-syntax-sanitization--06.mjml` | 42,203 | 70,069 | 688 | 26,377 | 43,793 | 430 |
| `upstream/mjml--template-syntax-sanitization--07.mjml` | 41,384 | 70,104 | 688 | 25,865 | 43,815 | 430 |
| `upstream/mjml--wrapper-border-radius--00.mjml` | 54,790 | 85,210 | 955 | 34,244 | 53,256 | 597 |
| `upstream/mjml--wrapper-gap--00.mjml` | 121,882 | 204,410 | 2,059 | 76,176 | 127,756 | 1,287 |
| `upstream/parser-xml--preprocessors--00.mjml` | 49,827 | 74,389 | 850 | 31,142 | 46,493 | 531 |

## Inline stage profile

The inline stage no longer round-trips the document through `html.Parse` and
`html.Render`. It tokenizes the markup the components emitted, matches the
selectors against that tree and rewrites only the start tags a rule changed
(`BenchmarkInlineStage`, document already rendered, medians of 5, measured on
this merged tree and on the pre-merge `main` in the same session):

| Fixture | Before ns/op | After ns/op | Before allocs/op | After allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `input_big.mjml` | 836,000 | 650,000 | 7,816 | 6,853 |
| `head.mjml` | 25,700 | 20,300 | 298 | 257 |

End to end, from the table above against the pre-merge numbers: `input_big.mjml`
2.50 ms -> 2.12 ms (26,090 -> 23,817 allocs/op) and
`head.mjml` 87.8 us -> 74.5 us (1,132 -> 1,062
allocs/op). Allocations drop on every fixture in the corpus; the few fixtures whose
time does not improve are the ones too small to measure the stage on (carousel,
accordion), where noise dominates.

That is ~0.19 ms of the ~0.54 ms the round-trip used to cost on `input_big.mjml`:
the rest of the round-trip's cost came back as the element tree the selector
passes need, and removing that too means components emitting elements instead of
markup (ADR-0005).

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
- The selector engine applies `+` and `~` in both directions, so `a + b` styles
  `a` as well. That is a correctness gap, not a budget one (ADR-0005).

`BenchmarkParse` and `BenchmarkParityComparison` isolate the parser and the test
harness.

## Reproducing

```bash
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=5
go test ./ -run XXX -bench BenchmarkInlineStage -benchmem -count=5
go test ./ -run XXX -bench BenchmarkParse -benchmem
go test ./ -run XXX -bench BenchmarkParityComparison -benchmem
```
