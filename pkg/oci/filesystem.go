package oci

// var (
// 	_ FilesystemProvider = &OSFilesystemProvider{}
// 	_ FilesystemProvider = &MemoryFilesystemProvider{}
// )

// // OSFilesystemProvider implements FilesystemProvider for the OS filesystem
// type OSFilesystemProvider struct {
// 	rootDir *os.Root
// }

// // NewOSFilesystemProvider creates a new OS filesystem provider
// func NewOSFilesystemProvider(rootDir string) (*OSFilesystemProvider, error) {
// 	root, err := os.OpenRoot(rootDir)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &OSFilesystemProvider{
// 		rootDir: root,
// 	}, nil
// }

// // ReadFile reads the file named by path and returns its contents
// func (p *OSFilesystemProvider) ReadFile(name string) ([]byte, error) {
// 	return fs.ReadFile(p.rootDir.FS(), name)
// }

// // WriteFile writes data to the named file, creating it if necessary
// func (p *OSFilesystemProvider) WriteFile(name string, data []byte, perm fs.FileMode) error {
// 	f, err := p.rootDir.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	_, err = io.Copy(f, bytes.NewReader(data))
// 	return err
// }

// func (p *OSFilesystemProvider) OpenRW(name string) (io.ReadWriteCloser, error) {
// 	return p.rootDir.OpenFile(name, os.O_RDWR, 0)
// }

// // MkdirAll creates a directory named path, along with any necessary parents
// func (p *OSFilesystemProvider) MkdirAll(path string, perm fs.FileMode) error {
// 	joined := filepath.Join(p.rootDir.Name(), path)
// 	return os.MkdirAll(joined, perm)
// }

// // RemoveAll removes path and any children it contains
// func (p *OSFilesystemProvider) RemoveAll(path string) error {
// 	fullPath := filepath.Join(p.rootDir.Name(), path)
// 	return os.RemoveAll(fullPath)
// }

// // Open opens the named file for reading
// func (p *OSFilesystemProvider) Open(name string) (fs.File, error) {
// 	return p.rootDir.Open(name)
// }

// // Stat returns file info for the named path
// func (p *OSFilesystemProvider) Stat(name string) (fs.FileInfo, error) {
// 	return p.rootDir.Stat(name)
// }

// // MemoryFilesystemProvider implements FilesystemProvider with an in-memory filesystem
// type MemoryFilesystemProvider struct {
// 	fs *mem.FS
// }

// // NewMemoryFilesystemProvider creates a new in-memory filesystem provider
// func NewMemoryFilesystemProvider() (*MemoryFilesystemProvider, error) {
// 	fs, err := mem.NewFS()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &MemoryFilesystemProvider{
// 		fs: fs,
// 	}, nil
// }

// func (p *MemoryFilesystemProvider) Open(name string) (fs.File, error) {
// 	return p.fs.Open(name)
// }

// // ReadFile reads the file named by path and returns its contents
// func (p *MemoryFilesystemProvider) ReadFile(name string) ([]byte, error) {
// 	f, err := p.fs.Open(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()

// 	return io.ReadAll(f)
// }

// // WriteFile writes data to the named file, creating it if necessary
// func (p *MemoryFilesystemProvider) WriteFile(name string, data []byte, perm fs.FileMode) error {
// 	dir := filepath.Dir(name)
// 	if dir != "." {
// 		if err := p.MkdirAll(dir, 0755); err != nil {
// 			return err
// 		}
// 	}

// 	// Create or truncate the file
// 	f, err := p.fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	// Write data using a bytes reader
// 	_, err = io.Copy(f.(io.Writer), bytes.NewReader(data))
// 	return err
// }

// // MkdirAll creates a directory named path, along with any necessary parents
// func (p *MemoryFilesystemProvider) MkdirAll(path string, perm fs.FileMode) error {
// 	return p.fs.MkdirAll(path, perm)
// }

// // RemoveAll removes path and any children it contains
// func (p *MemoryFilesystemProvider) RemoveAll(path string) error {
// 	if path == "" {
// 		// Special case for removing everything
// 		newFs, _ := mem.NewFS()
// 		*p.fs = *newFs
// 		return nil
// 	}

// 	// For standard paths, try to remove the entry
// 	// Note: This is a simplistic implementation as mem.FS doesn't have a true RemoveAll
// 	return p.fs.Remove(path)
// }

// // Open opens the named file for reading
// func (p *MemoryFilesystemProvider) OpenRW(name string) (io.ReadWriteCloser, error) {
// 	f, err := p.fs.OpenFile(name, os.O_RDWR, 0)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return f.(io.ReadWriteCloser), nil
// }

// // Stat returns file info for the named path
// func (p *MemoryFilesystemProvider) Stat(name string) (fs.FileInfo, error) {
// 	return p.fs.Stat(name)
// }
