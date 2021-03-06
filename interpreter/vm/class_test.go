package vm_test

import (
	"os"
	"path/filepath"

	. "github.com/grubby/grubby/interpreter/vm"
	. "github.com/grubby/grubby/interpreter/vm/builtins"
	. "github.com/grubby/grubby/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("classes", func() {
	var vm VM

	BeforeEach(func() {
		pathToExecutable, err := filepath.Abs(filepath.Dir(filepath.Dir(filepath.Dir(os.Args[0]))))
		if err != nil {
			panic(err)
		}

		vm = NewVM(pathToExecutable, "fake-irb-under-test")
	})

	It("can be used as a value", func() {
		value, err := vm.Run(`
class Foo::Bar
end

foo = Foo::Bar
`)

		Expect(err).ToNot(HaveOccurred())
		Expect(value).ToNot(BeNil())
		Expect(value).To(Equal(vm.MustGetClass("Foo::Bar")))
	})

	Describe(".new", func() {
		It("returns an error when initializing the object would fail", func() {
			_, err := vm.Run(`
class Microclimatology
  def initialize
    overchildish
  end
end

Microclimatology.new
`)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("NameError"))
			Expect(err.Error()).To(ContainSubstring("undefined local variable or method 'overchildish'"))
		})
	})

	Describe("class attribute methods", func() {
		Describe(".attr_reader :symbol", func() {
			It("creates a getter and setter on instances of the class", func() {
				_, err := vm.Run(`
class Foo
  attr_reader :quaternion_vinic
end
`)

				Expect(err).ToNot(HaveOccurred())

				foo, err := vm.MustGetClass("Foo").New(vm, vm)
				Expect(err).ToNot(HaveOccurred())

				reader, err := foo.Method("quaternion_vinic")
				Expect(err).ToNot(HaveOccurred())

				val, err := reader.Execute(foo, nil)
				Expect(err).ToNot(HaveOccurred())

				nilInstance := vm.SingletonWithName("nil")

				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(nilInstance))
			})
		})

		Describe(".attr_writer :some_symbol", func() {
			It("creates a setter on instances of the class", func() {
				_, err := vm.Run(`
class Foo
  attr_writer :chrysobull_nonmonarchist
end
`)

				Expect(err).ToNot(HaveOccurred())

				fooClass := vm.MustGetClass("Foo")
				foo, err := fooClass.New(vm, vm)
				Expect(err).ToNot(HaveOccurred())

				reader, err := foo.Method("chrysobull_nonmonarchist=")
				Expect(err).ToNot(HaveOccurred())

				_, err = reader.Execute(foo, nil, NewString("lyncher-mudslinger", vm, vm))
				Expect(err).ToNot(HaveOccurred())

				// TODO: assert on the instance variable via instance_variable_get
			})
		})

		Describe(".attr_accessor :some_symbol", func() {
			It("creates a getter and setter on instances of the class", func() {
				_, err := vm.Run(`
class Foo
  attr_accessor :pieless_bothlike
end
`)

				Expect(err).ToNot(HaveOccurred())

				fooClass := vm.MustGetClass("Foo")
				foo, err := fooClass.New(vm, vm)
				Expect(err).ToNot(HaveOccurred())

				reader, err := foo.Method("pieless_bothlike")
				Expect(err).ToNot(HaveOccurred())

				val, err := reader.Execute(foo, nil)
				Expect(err).ToNot(HaveOccurred())

				nilInstance := vm.SingletonWithName("nil")
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(nilInstance))

				writer, err := foo.Method("pieless_bothlike=")
				Expect(err).ToNot(HaveOccurred())

				_, err = writer.Execute(foo, nil, NewString("unordainable-luthier", vm, vm))
				Expect(err).ToNot(HaveOccurred())

				val, err = reader.Execute(foo, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(EqualRubyString("unordainable-luthier"))
			})
		})
	})

	Describe("private methods", func() {
		It("can be created from a public method using .private_class_method()", func() {
			class, err := vm.Run(`
class Foo
  def self.bar
  end

  private_class_method :bar
end
`)

			Expect(err).ToNot(HaveOccurred())

			_, err = class.PrivateMethod("bar")
			Expect(err).ToNot(HaveOccurred())

			Expect(class).ToNot(HaveMethod("bar"))
		})
	})

	Describe("superclasses", func() {
		It("defaults to Object", func() {
			class, err := vm.Run(`
class Foo
end
`)
			Expect(err).ToNot(HaveOccurred())
			Expect(class.(Class).SuperClass().String()).To(Equal("Object"))
		})
	})

	It("is a kind of module", func() {
		classClass := vm.MustGetClass("Class")
		Expect(classClass.(Class).SuperClass().String()).To(Equal("Module"))

		_, ok := classClass.(Module)
		Expect(ok).To(BeTrue())
	})
})
