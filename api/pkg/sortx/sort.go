package sortx

func Int64(data []int64) []int64 {
	if len(data) <= 1 {
		return data
	}

	return sortInt64(data)
}

func sortInt64(data []int64) []int64 {
	if len(data) <= 1 {
		return data
	}

	if len(data) == 2 {
		if data[0] > data[1] {
			data[0], data[1] = data[1], data[0]
		}

		return data
	}

	//{5,6,3}
	mid := data[len(data)/2]      //6
	var little = make([]int64, 0) //
	var large = make([]int64, 0)
	for _, val := range data {
		if val < mid {
			little = append(little, val)
		}
		if val > mid {
			large = append(large, val)
		}
	}

	little = sortInt64(little)
	large = sortInt64(large)

	little = append(little, mid)

	data = append(little, large...)
	return data
}
