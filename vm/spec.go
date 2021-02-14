package vm

func initSpecClass(vm *VM) {
	// TODO: Could we embed this in the binary?
	_ = vm.newThread().loadLibrary("spec.lito")
}
