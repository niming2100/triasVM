package util

func StringArrayToByte(strArray []string) [][]byte{
	var args [][]byte;
	for _,v:= range strArray{
		args = append(args,[]byte(v));
	}
	return args;
}