.PHONY: deepfuzz quantfuzz

# a Makefile for fuzzing, since remembering the exact
# commands is tricky.

deepfuzz:
	go test -timeout=0 -count=1 -fuzz=FuzzARTDeep -run=blah |tee fuzz.deep.log

quantfuzz:
	go test -timeout=0 -count=1 -fuzz=FuzzAscendDescend -run=blah |tee fuzz.ad.log

fuzz0:
	go test -timeout=0 -count=1 -fuzz=FuzzNoPrefixAscendDescend -run=blah |tee fuzz.0.log

