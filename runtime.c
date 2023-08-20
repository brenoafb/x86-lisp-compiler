#include <stdio.h>
#include <stdlib.h>

#define empty_list      0x2f

#define fixnum_mask     3
#define fixnum_tag      0
#define fixnum_shift    2

#define char_mask       0xff
#define char_tag        0x0f
#define char_shift      8

#define bool_mask		0x7f
#define bool_tag		0x1f
#define bool_shift		7

#define ptr_mask        7
#define pair_tag        1
#define vector_tag      2
#define string_tag      3
#define symbol_tag      5
#define closure_tag     6

#define HEAPSIZE        1024 * 1024

int entry(void *heap);

int main(int argc, char *argv[]) {
	void *heap = malloc(HEAPSIZE);
	printf("heap: 0x%x\n", heap);
	int val = entry(heap);
	printf("0x%x\n", val);
	if ((val & fixnum_mask) == fixnum_tag) {
		printf("%d\n", val >> fixnum_shift);
	} else if ((val & char_mask) == char_tag) {
		printf("%c\n", (char) (val >> char_shift));
	} else if (val == empty_list) {
		printf("()\n");
	} else if ((val & bool_mask) == bool_tag) {
		int shifted = val >> bool_shift;
		printf("#%c\n", (val >> bool_shift) ? 't' : 'f');
	} else if ((val & ptr_mask) == pair_tag) {
		printf("#<pair 0x%x>\n", val);
	} else if ((val & ptr_mask) == vector_tag) {
		printf("#<vector 0x%x>\n", val);
	} else if ((val & ptr_mask) == string_tag) {
		printf("#<string 0x%x>\n", val);
		printf("%s\n", val & ~3);
	} else if ((val & ptr_mask) == symbol_tag) {
		printf("#<symbol 0x%x>\n", val);
	} else if ((val & ptr_mask) == closure_tag) {
		printf("#<closure 0x%x>\n", val);
	} else {
		printf("#<unknown 0x%x>\n", val);
	}

	return 0;
}
