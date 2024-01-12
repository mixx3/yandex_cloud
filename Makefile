.PHONY:
	test

ZONE_ID ?= ""
ZONE_NAME ?= ""
IAM_TOKEN ?= ""

test:
	ZONE_ID=$(ZONE_ID) \
	ZONE_NAME=$(ZONE_NAME) \
	IAM_TOKEN=$(IAM_TOKEN) \
	go test -v . | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''
