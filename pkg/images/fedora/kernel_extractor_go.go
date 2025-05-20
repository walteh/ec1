package fedora

// // ExtractKernelGo extracts a Linux kernel from an EFI application if necessary
// // using the pure Go implementation of unzboot.
// // This is useful for handling the case where the kernel is embedded in an EFI application.
// func (prov *FedoraProvider) ExtractKernelGo(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {
// 	// Check if kernel is an EFI zboot image
// 	kernelReader, ok := cacheDir["kernel"]
// 	if !ok {
// 		return cacheDir, nil // No kernel to process
// 	}

// 	// Read the extracted kernel
// 	extractedData, err := os.ReadFile(extractedKernel)
// 	if err != nil {
// 		return nil, errors.Errorf("reading extracted kernel: %w", err)
// 	}

// 	// Replace the kernel in the cache with the extracted one
// 	cacheDir["kernel"] = bytes.NewReader(extractedData)

// 	return cacheDir, nil
// }
