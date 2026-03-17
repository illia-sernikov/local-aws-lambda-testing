BUILD_DIR := lambda/build
LAMBDA_BIN := $(BUILD_DIR)/bootstrap
LAMBDA_ZIP := $(BUILD_DIR)/lambda.zip
LAMBDA_DIR := lambda
PULUMI_DIR := deploy/pulumi

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: build-lambda package-lambda clean infra-up deploy destroy logs

build-lambda:
	mkdir -p $(BUILD_DIR)
	cd $(LAMBDA_DIR) && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(LAMBDA_BIN) ./cmd/handler

package-lambda: build-lambda
	zip -j $(LAMBDA_ZIP) $(LAMBDA_BIN)

clean:
	rm -rf $(BUILD_DIR) build

infra-up:
	cd $(PULUMI_DIR) && pulumi login --local && pulumi up --yes

deploy: package-lambda infra-up

destroy:
	cd $(PULUMI_DIR) && pulumi login --local && pulumi destroy --yes

logs:
	awslocal logs tail /aws/lambda/handler --follow
