
all:
	go build

clean:
	./cleanup-prev-test.sh
	rm -f testdata/pipe1 testdata/pipe2 testdata/pipe3 ./log-consolidate consolidate-log.log testdata/test1.out.log

test: test1 test2

test1:
	@rm -f ./log-consolidate testdata/test1.out.log
	@go build
	@./cleanup-prev-test.sh						# zap previous tests 
	@rm -f testdata/pipe2
	@echo start reader							# start reader/writer
	./log-consolidate --cfg testdata/test1-read.json read &	
	./log-consolidate --cfg testdata/test1-write.json write &
	@sleep 1
	@echo AA >testdata/pipe1					# send data thorugh pipes
	@echo BB >testdata/pipe2
	@echo Aa >testdata/pipe1
	@echo Ba >testdata/pipe2
	@echo Ac >testdata/pipe1
	@echo Ad >testdata/pipe1
	@echo Ae >testdata/pipe1
	@echo Af >testdata/pipe1
	@echo Bb >testdata/pipe2
	@echo CC >testdata/pipe3
	@echo Ca >testdata/pipe3
	@ls -l testdata/pipe1 >/dev/null			# Check that the pipes are there
	@ls -l testdata/pipe2 >/dev/null
	@ls -l testdata/pipe3 >/dev/null
	@sleep 1
	@grep AA testdata/test1.out.log >/dev/null	# Check that the output has data it should
	@grep BB testdata/test1.out.log >/dev/null
	@grep CC testdata/test1.out.log >/dev/null
	@./cleanup-prev-test.sh						# zap running tests 
	@echo PASS									# if we get to this point then it is a PASS


test2:
	( cd ./lib ; go test )

