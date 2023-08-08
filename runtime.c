#include <stdio.h>

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

int scheme_entry();

int main(int argc, char *argv[]) {
	int val = scheme_entry();
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
	} else {
		printf("#<unknown 0x%x>\n", val);
	}

	return 0;
}
