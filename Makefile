.PHONY: build run clean

# Nombre del binario de salida
BINARY_NAME=fancy_welcome

build:
	@echo "Construyendo el binario..."
	@go build -o $(BINARY_NAME) .

run:
	@go run .

clean:
	@echo "Limpiando..."
	@go clean
	@rm -f $(BINARY_NAME)
