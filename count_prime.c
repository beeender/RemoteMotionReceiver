#include "postgres.h"
#include "fmgr.h"
#include "utils/builtins.h"
#include <math.h>

PG_MODULE_MAGIC;

PG_FUNCTION_INFO_V1(count_prime);
Datum
count_prime(PG_FUNCTION_ARGS)
{
    int num1 = 0;
    int num2 = PG_GETARG_INT32(0);
    int count = 0;

	if (num2 < 2) {
		return count;
	}
	while (num1 <= num2) {
		bool isPrime = true;
		for (int i = 2; i <= (int)sqrt(num1); i++) {
			if (num1%i == 0) {
				isPrime = false;
				break;
			}
		}
		if (isPrime) count++;
		num1++;
	}

    PG_RETURN_INT32(count);
}


