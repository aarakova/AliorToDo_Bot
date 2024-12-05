MOCKS_DESTINATION=mocks
INTERFACE_FILES=src/tg_provider/tg.go

.PHONY: mocks clean

# Генерация моков
mocks: $(INTERFACE_FILES)
	@echo "Generating mocks..."
	@if exist $(MOCKS_DESTINATION) (rmdir /s /q $(MOCKS_DESTINATION))
	@mkdir $(MOCKS_DESTINATION)
	@for %%f in ($(INTERFACE_FILES)) do ( \
		mockgen -source=%%f -destination=$(MOCKS_DESTINATION)/%%~nf_mock.go -package=tg_provider \
	)

# Очистка сгенерированных файлов
clean:
	@echo "Cleaning mocks..."
	@if exist $(MOCKS_DESTINATION) (rmdir /s /q $(MOCKS_DESTINATION))
