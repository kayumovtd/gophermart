package ordervalidation

// IsValidOrderNumber проверяет номер заказа на соответствие формату и алгоритму Луна.
func IsValidOrderNumber(number string) bool {
	if number == "" {
		return false
	}

	for i := 0; i < len(number); i++ {
		if number[i] < '0' || number[i] > '9' {
			return false
		}
	}

	return luhnValid(number)
}

func luhnValid(number string) bool {
	sum := 0
	doubleNext := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if doubleNext {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		doubleNext = !doubleNext
	}

	return sum%10 == 0
}
