package xval

import "testing"

const consoleSerialNumber = "031897772807"

func TestThatConsoleIsBanned(t *testing.T) {
	xval := "DA9C-DC84-43A9-BB4C"

	_, decryptedXVal, err := Decrypt(consoleSerialNumber, xval)

	if err != nil {
		t.Error(err)
		return
	}

	xvalDescriptions := TextResult(decryptedXVal)

	if len(xvalDescriptions) != 1 {
		t.Error("Xval should not be clean")
	} else if len(xvalDescriptions) > 1 {
		t.Error("Xval should only have the console ban flag")
	} else if bannedDescription := getFlagDescriptions()[FlagConsoleBanned]; xvalDescriptions[0] != bannedDescription {
		t.Errorf("Expecting xval description result \"%s\", got \"%s\"", bannedDescription, xvalDescriptions[0])
	}
}
