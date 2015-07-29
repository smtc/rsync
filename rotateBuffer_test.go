package rsync

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

var (
	ss []string = []string{
		"",
		"1",
		"12",
		"123",
		"1234",
		"12345",
		"123456",
		"1234567",
		"12345678",
		"123456789",
		"1234567890",
		"12345678901",
		"123456789012",
		"1234567890123",
		"12345678901234",
		"123456788765432",
		"1234567887654321",
		"12345678876543210",
		"123456788765432101",
		"1234567887654321012",
		"12345678876543210123",
		"123456788765432101234",
		"1234567887654321012345",
		"12345678876543210123456",
		"123456788765432101234567",
		"1234567887654321012345678",
		"12345678876543210123456787",
		"123456788765432101234567876",
		"1234567887654321012345678765",
		"12345678876543210123456787654",
		"123456788765432101234567876543",
		"1234567887654321012345678765432",
		"12345678876543210123456787654321",
		"123456788765432101234567876543210",
		"1234567887654321012345678765432101",
		"12345678876543210123456787654321012",
		"123456788765432101234567876543210123",
		"1234567887654321012345678765432101234",
		"12345678876543210123456787654321012345",
		"123456788765432101234567876543210123456",
		"1234567887654321012345678765432101234567",
		"12345678876543210123456787654321012345678",
		"123456788765432101234567876543210123456787",
		"1234567887654321012345678765432101234567876",
		"12345678876543210123456787654321012345678765",
		"123456788765432101234567876543210123456787654",
		"1234567887654321012345678765432101234567876543",
		"12345678876543210123456787654321012345678765432",
		"123456788765432101234567876543210123456787654321",
		"1234567887654321012345671234567887654321012345678",
		"12345678876543210123456712345678876543210123456789",
		"123456788765432101234567123456788765432101234567890",
	}
)

func testRotateBuffer(t *testing.T) {
	for i, s := range ss {
		testRotate(t, i, 8, s)
	}
}

func testRotate(t *testing.T, i, blockLen int, s string) {
	t.Log(i, s)
	testRollByte(t, i, blockLen, s)
	testRollBlock(t, i, blockLen, s)
	testRollHybrid(t, i, blockLen, s)
	testRollRandom(t, i, blockLen, s)
}

func testRollByte(t *testing.T, idx, blockLen int, s string) {
	var c byte
	readed := 0
	rb := NewRotateBuffer(int64(len(s)), blockLen, bytes.NewBufferString(s))
	p, initial, err := rb.rollFirst()
	readed += len(p)
	Assert(initial == 0, "initial must be 0 in first rollByte")
	if len(s) == 0 {
		Assert(err == noBytesLeft, "empty string should return noBytesLeft in first rollByte")
	} else if len(s) < 8 {
		readed = 0
		Assertf(err == notEnoughBytes,
			"string less than blockLen(8) return notEnoughBytes in first read. idx: %d string: %s",
			idx, s)
	} else {
		Assertf(readed == int(rb.absTail), "readed %d should equal with absTail %d", readed, rb.absTail)
	}

	t.Log("first read:", string(p))
	for err == nil {
		if p, c, initial, err = rb.rollByte(); err == nil {
			t.Log("  rollByte:", string(c), initial, string(p))
			Assertf(initial > 0,
				"initial should > 0 in following roll. string: %d %s",
				idx, s)
			Assertf(c == byte(s[readed-len(p)]),
				"read ch is %v, but should be %s, readed: %d. idx: %d string: %s",
				string(c), string(s[readed]), readed, idx, s)
			readed++
		}
	}
	t.Logf("RollByte: start=%d end=%d absHead=%d absEnd=%d err=%v\n",
		rb.start, rb.end, rb.absHead, rb.absTail, err)

	if err == notEnoughBytes {
		err = nil
		t.Log(" RollByte: rollLeft: eof:", rb.eof)
		for err == nil {
			if p, c, initial, err = rb.rollLeft(); err == nil {
				t.Log("  RollByte: rollLeft:", string(c), initial, string(p))
				readed++
			}
		}
	}

	//Assertf(readed == len(s),
	//	"readed(%d) should equal with string length(%d) when read complete. idx: %d string: %s",
	//	readed, len(s), idx, s)
	Assertf(int(rb.absTail) == len(s), "abs tail should be len(s) when read complete.")
}

func testRollBlock(t *testing.T, idx, blockLen int, s string) {
	var c byte
	readed := 0
	rb := NewRotateBuffer(int64(len(s)), 8, bytes.NewBufferString(s))
	p, initial, err := rb.rollFirst()
	readed += len(p)
	Assert(initial == 0, "initial must be 0 in first rollByte")
	if len(s) == 0 {
		Assert(err == noBytesLeft, "empty string should return noBytesLeft in first rollByte")
	} else if len(s) < 8 {
		Assertf(err == notEnoughBytes,
			"string less than blockLen(8) return notEnoughBytes in first read. idx: %d string: %s",
			idx, s)
	} else {
		Assert(readed == int(rb.absTail), "readed should equal with absTail")
	}
	if err == nil {
		_ = c
		//Assertf(c == s[0], "first byte %v %v error. idx: %d string: %s", c, s[0], idx, s)
	}
	for err == nil {
		if p, initial, err = rb.rollBlock(); err == nil {
			t.Log("  rollBlock:", initial, string(p))
			Assertf(len(p) == rb.blockLen, "readBlock return slice lenght should be blockLen")
			readed += len(p)
		}
	}
	t.Logf("RollBlock: start=%d end=%d absHead=%d absEnd=%d err=%v\n",
		rb.start, rb.end, rb.absHead, rb.absTail, err)

	if err == notEnoughBytes {
		err = nil
		for err == nil {
			if p, c, initial, err = rb.rollLeft(); err == nil {
				t.Log("  RollBlock: rollLeft:", string(c), initial, string(p))
				readed++
			}
		}
	}
	//Assertf(readed == len(s),
	//	"readed(%d) should equal with string length(%d) when read complete. idx: %d string: %s",
	//	readed, len(s), idx, s)
	Assertf(int(rb.absTail) == len(s), "abs tail should be len(s) when read complete.")
}

//
func testRollHybrid(t *testing.T, idx, blockLen int, s string) {
	var c byte
	readed := 0
	rb := NewRotateBuffer(int64(len(s)), 8, bytes.NewBufferString(s))
	p, initial, err := rb.rollFirst()
	readed += len(p)
	Assert(initial == 0, "initial must be 0 in first rollByte")
	if len(s) == 0 {
		Assert(err == noBytesLeft, "empty string should return noBytesLeft in first rollByte")
	} else if len(s) < 8 {
		Assertf(err == notEnoughBytes,
			"string less than blockLen(8) return notEnoughBytes in first read. idx: %d string: %s",
			idx, s)
	} else {
		Assert(readed == int(rb.absTail), "readed should equal with abs tail")
	}
	if err == nil {
		_ = c
		//Assertf(c == s[0], "first byte %v %v error. idx: %d string: %s", c, s[0], idx, s)
	}
	for err == nil {
		if p, c, initial, err = rb.rollByte(); err == nil {
			t.Log("  RollHybrid: rollByte:", string(c), initial, string(p))
			Assertf(initial > 0,
				"initial should be false in following roll. string: %d %s",
				idx, s)
			Assertf(c == byte(s[readed-len(p)]),
				"read ch is %v, but should be %s, readed: %d. idx: %d string: %s",
				string(c), string(s[readed]), readed, idx, s)
			readed++
		}
		if p, initial, err = rb.rollBlock(); err == nil {
			t.Log("  RollHybrid: rollBlock:", initial, string(p))
			Assertf(len(p) == rb.blockLen, "readBlock return slice lenght should be blockLen")
			readed += len(p)
		}
	}
	t.Logf("start=%d end=%d absHead=%d absEnd=%d err=%v\n", rb.start, rb.end, rb.absHead, rb.absTail, err)

	if err == notEnoughBytes {
		err = nil
		for err == nil {
			if p, c, initial, err = rb.rollLeft(); err == nil {
				t.Log("  RollHybrid: rollLeft:", string(c), initial, string(p))
				readed++
			}
		}
	}
	//Assertf(readed == len(s),
	//	"readed(%d) should equal with string length(%d) when read complete. idx: %d string: %s",
	//	readed, len(s), idx, s)
	Assertf(int(rb.absTail) == len(s), "tail should be len(s) when read complete.")

}

// 随机调用rollByte和rollBlock
func testRollRandom(t *testing.T, idx, blockLen int, s string) {
	var c byte
	readed := 0
	rb := NewRotateBuffer(int64(len(s)), 8, bytes.NewBufferString(s))
	p, initial, err := rb.rollFirst()
	readed += len(p)
	Assert(initial == 0, "initial must be 0 in first rollByte")
	if len(s) == 0 {
		Assert(err == noBytesLeft, "empty string should return noBytesLeft in first rollByte")
	} else if len(s) < 8 {
		Assertf(err == notEnoughBytes,
			"string less than blockLen(8) return notEnoughBytes in first read. idx: %d string: %s",
			idx, s)
	} else {
		Assert(readed == int(rb.absTail), "readed should equal with absTail")
	}
	if err == nil {
		_ = c
		//Assertf(c == s[0], "first byte %v %v error. idx: %d string: %s", c, s[0], idx, s)
	}
	rand.Seed(time.Now().Unix())
	for err == nil {
		r := rand.Intn(2)
		if r == 0 {
			if p, c, initial, err = rb.rollByte(); err == nil {
				t.Log("  RollRandom: rollByte:", string(c), initial, string(p))
				Assertf(initial > 0,
					"initial should be false in following roll. string: %d %s",
					idx, s)
				Assertf(c == byte(s[readed-len(p)]),
					"read ch is %v, but should be %s, readed: %d. idx: %d string: %s",
					string(c), string(s[readed]), readed, idx, s)
				readed++
			}
		} else {
			if p, initial, err = rb.rollBlock(); err == nil {
				t.Log(" RollRandom: rollBlock:", initial, string(p))
				Assertf(len(p) == rb.blockLen, "readBlock return slice lenght should be blockLen")
				readed += len(p)
			}
		}
	}
	t.Logf(" RollRandom: start=%d end=%d absHead=%d absEnd=%d err=%v\n", rb.start, rb.end, rb.absHead, rb.absTail, err)

	if err == notEnoughBytes {
		err = nil
		for err == nil {
			if p, c, initial, err = rb.rollLeft(); err == nil {
				t.Log(" RollRandom: rollLeft:", string(c), initial, string(p))
				readed++
			}
		}
	}
	//Assertf(readed == len(s),
	//	"readed(%d) should equal with string length(%d) when read complete. idx: %d string: %s",
	//	readed, len(s), idx, s)
	Assertf(int(rb.absTail) == len(s), "left should be 0 when read complete.")
}
