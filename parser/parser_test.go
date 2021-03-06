package parser_test

import (
	"fmt"

	"github.com/grubby/grubby/ast"
	"github.com/grubby/grubby/parser"

	. "github.com/grubby/grubby/parser/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("goyacc parser", func() {
	var (
		lexer parser.RubyLexer
	)

	JustBeforeEach(func() {
		parser.DebugStatements = []string{}
		parser.Statements = make([]ast.Node, 0)
	})

	Describe("parsing nothing at all", func() {
		It("succeeds without any errors", func() {
			lexer = parser.NewLexer("")
			Expect(parser.RubyParse(lexer)).To(BeSuccessful())
			Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).ToNot(HaveOccurred())
		})
	})

	Context("when the code parsed is syntactically valid", func() {
		JustBeforeEach(func() {
			Expect(parser.RubyParse(lexer)).To(BeSuccessful())
			Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).ToNot(HaveOccurred())
		})

		Describe("parsing an integer", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("5")
			})

			It("returns a ConstantInt struct representing the value", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.ConstantInt{Value: 5},
				}))
			})
		})

		Describe("parsing a float", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("123.4567")
			})

			It("returns a ConstantFloat struct representing the value", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.ConstantFloat{Value: 123.4567},
				}))
			})
		})

		Describe("backtics", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("`echo 'this wont work on windows'`")
			})

			It("returns a Subshell struct", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Subshell{Command: "echo 'this wont work on windows'"},
				}))
			})
		})

		Describe("strings", func() {
			Context("with single quotes", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("'hello world'")
				})

				It("returns a SimpleString struct", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.SimpleString{Value: "hello world"},
					}))
				})
			})

			Context("with escaped single quotes", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("'hello \\' world'")
				})

				It("returns a SimpleString struct", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.SimpleString{Value: "hello \\' world"},
					}))
				})
			})

			Context("with double quotes", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`"pianic-#{foo}-vespid"`)
				})

				It("returns a InterpolatedString struct", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.InterpolatedString{Value: "pianic-#{foo}-vespid"},
					}))
				})

				Context("with escaped double quotes", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`"pianic-\"-vespid"`)
					})

					It("returns a InterpolatedString struct", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.InterpolatedString{Value: `pianic-\"-vespid`},
						}))
					})
				})

				Context("with string interpolation inside string interpolation", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
"#{@tag}#{ "(#{@comment})" if @comment }:#{escape @description}"
`)
					})

					It("returns an interpolated string", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.InterpolatedString{
								Value: `#{@tag}#{ "(#{@comment})" if @comment }:#{escape @description}`,
							},
						}))
					})
				})

				Context("with double quotes inside the interpolation", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`"Raj-#{5 * " "}-Corin"`)
					})

					It("returns a InterpolatedString struct", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.InterpolatedString{Value: `Raj-#{5 * " "}-Corin`},
						}))
					})
				})
			})

			Describe("ascii character literals", func() {
				for asciiValue := 33; asciiValue <= 126; asciiValue++ {
					func(ascii int) {
						Context(fmt.Sprintf("?%s", string(ascii)), func() {
							BeforeEach(func() {
								lexer = parser.NewLexer("?" + string(ascii))
							})

							It("parses as a character literal", func() {
								Expect(parser.Statements).To(Equal([]ast.Node{
									ast.CharacterLiteral{Value: string(ascii)},
								}))
							})
						})
					}(asciiValue)
				}
			})

			Describe("heredoc", func() {
				Context("as an arg to a method call", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
foo(<<-EOS, 'a', 'b', 'c', __LINE__ + 1)
something
goes
here
EOS
`)
					})

					It("should parse the heredoc around the method call", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Func: ast.BareReference{Name: "foo"},
								Args: []ast.Node{
									ast.InterpolatedString{Value: "something\ngoes\nhere"},
									ast.SimpleString{Value: "a"},
									ast.SimpleString{Value: "b"},
									ast.SimpleString{Value: "c"},
									ast.CallExpression{
										Target: ast.LineNumberConstReference{},
										Func:   ast.BareReference{Name: "+"},
										Args:   []ast.Node{ast.ConstantInt{Value: 1}},
									},
								},
							},
						}))
					})
				})

				Context("with a dash", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
<<-FOO
spheniscomorphic-monoptic
   FOO
`)
					})

					It("returns an interpolated string, ending when it finds the identifier", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.InterpolatedString{Value: "spheniscomorphic-monoptic"},
						}))
					})
				})

				Context("without dash", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
<<FOO
resenter-postvenous
  FOO
FOO
`)
					})

					It("returns an interpolated string, ending only when it finds the identifier at column 0", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.InterpolatedString{Value: "resenter-postvenous\n  FOO"},
						}))
					})
				})
			})

			Context("with % notation", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
%[ain't life grand]
%<this is the worst>
%{GET OUT}
%(DONT EVER WRITE CODE LIKE THIS)
`)
				})

				It("is parsed as an interpolated string", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.InterpolatedString{Value: "ain't life grand"},
						ast.InterpolatedString{Value: "this is the worst"},
						ast.InterpolatedString{Value: "GET OUT"},
						ast.InterpolatedString{Value: "DONT EVER WRITE CODE LIKE THIS"},
					}))
				})
			})
		})

		Describe("symbols", func() {
			Describe("generated dynamically with strings using interpolation", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`instance_variable_get :"@#{symbol}"`)
				})

				It("is parsed as a regular Symbol with the interpolation template still present", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "instance_variable_get"},
							Args: []ast.Node{ast.Symbol{Name: "@#{symbol}"}},
						},
					}))
				})
			})

			Context("without any special characters", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(":foo")
				})

				It("returns an ast.Symbol", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Symbol{Name: "foo"},
					}))
				})
			})

			Context("with an @ following the :", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(":@foo")
				})

				It("is parsed just like a regular symbol", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{ast.Symbol{Name: "@foo"}}))
				})
			})

			Context("with special characters", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(":foo!bar?")
				})

				It("is parsed as a symbol", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Symbol{Name: "foo!bar?"},
					}))
				})
			})
		})

		Describe("parsing multiple lines", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
:foo
:bar`)
			})

			It("returns multiple nodes", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Symbol{Name: "foo"},
					ast.Symbol{Name: "bar"},
				}))
			})
		})

		Describe("variables", func() {
			Context("that the user has defined", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("foo")
				})

				It("returns a bare reference", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.BareReference{Name: "foo"},
					}))
				})
			})

			Context("__FILE__", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("__FILE__")
				})

				It("returns a file name reference", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FileNameConstReference{},
					}))
				})
			})

			Context("__LINE__", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("__LINE__")
				})

				It("returns a line number reference", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.LineNumberConstReference{},
					}))
				})

				Describe("as an object", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer("__LINE__ + 1")
					})

					It("can have methods called on it", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Target: ast.LineNumberConstReference{},
								Func:   ast.BareReference{Name: "+"},
								Args:   []ast.Node{ast.ConstantInt{Value: 1}},
							},
						}))
					})
				})
			})
		})

		Describe("case statements", func() {
			Context("without a value to compare against", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
case
  when truthy_method?
    0
  when falsey_method?
    1
end
`)
				})

				It("should be parsed as an ast.SwitchStatement", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.SwitchStatement{
							Cases: []ast.SwitchCase{
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "truthy_method?"},
										},
									},
									Body: []ast.Node{ast.ConstantInt{Value: 0}},
								},
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "falsey_method?"},
										},
									},
									Body: []ast.Node{ast.ConstantInt{Value: 1}},
								},
							},
						},
					}))
				})
			})

			Context("with a value to compare against", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
case some_integer
when 1, 3
  puts 'even'
when 2, 4
  puts 'odd'
when ?^
  puts 'a single character (^) literal'
when ?:
  puts 'a single character (:) literal'
else
  puts 'whoops'
end
`)
				})

				It("should be parsed as an ast.SwitchStatement", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.SwitchStatement{
							Condition: ast.BareReference{Name: "some_integer"},
							Cases: []ast.SwitchCase{
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.ConstantInt{Value: 1},
										ast.ConstantInt{Value: 3},
									},
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "even"}},
										},
									},
								},
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.ConstantInt{Value: 2},
										ast.ConstantInt{Value: 4},
									},
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "odd"}},
										},
									},
								},
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.CharacterLiteral{Value: "^"},
									},
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "a single character (^) literal"}},
										},
									},
								},
								ast.SwitchCase{
									Conditions: []ast.Node{
										ast.CharacterLiteral{Value: ":"},
									},
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "a single character (:) literal"}},
										},
									},
								},
							},
							Else: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{
										ast.SimpleString{Value: "whoops"},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("procs", func() {
			Context("created with curly braces", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("Proc.new { Mock.verify_count }")
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "Proc"},
							Func:   ast.BareReference{Name: "new"},
							Args:   []ast.Node{},
							OptionalBlock: ast.Block{
								Body: []ast.Node{
									ast.CallExpression{
										Target: ast.BareReference{Name: "Mock"},
										Func:   ast.BareReference{Name: "verify_count"},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("call expressions", func() {
			Context("with a value that should be converted to a proc", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("describe(&blocks); explain(&:it_well)")
				})

				It("converts the argument to a proc", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "describe"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "blocks"},
									Func:   ast.BareReference{Name: "to_proc"},
								},
							},
						},
						ast.CallExpression{
							Func: ast.BareReference{Name: "explain"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.Symbol{Name: "it_well"},
									Func:   ast.BareReference{Name: "to_proc"},
								},
							},
						},
					}))
				})
			})

			Context("with inline assignment of binary operators", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
a += 5
b -= [1]
c /= 'x'
d *= nil
`)
				})

				It("is parsed with the operator as the method name", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "a"},
							Func:   ast.BareReference{Name: "+="},
							Args:   []ast.Node{ast.ConstantInt{Value: 5}},
						},
						ast.CallExpression{
							Target: ast.BareReference{Name: "b"},
							Func:   ast.BareReference{Name: "-="},
							Args: []ast.Node{ast.Array{
								Nodes: []ast.Node{ast.ConstantInt{Value: 1}},
							}},
						},
						ast.CallExpression{
							Target: ast.BareReference{Name: "c"},
							Func:   ast.BareReference{Name: "/="},
							Args:   []ast.Node{ast.SimpleString{Value: "x"}},
						},
						ast.CallExpression{
							Target: ast.BareReference{Name: "d"},
							Func:   ast.BareReference{Name: "*="},
							Args:   []ast.Node{ast.Nil{}},
						},
					}))
				})
			})

			Context("with arguments split across newlines", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
method_with_lots_of_args('foo',
                         'bar',
                         'baz',
                         &buz)
`)
				})

				It("should be parsed as though the newlines were not present", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "method_with_lots_of_args"},
							Args: []ast.Node{
								ast.SimpleString{Value: "foo"},
								ast.SimpleString{Value: "bar"},
								ast.SimpleString{Value: "baz"},
								ast.CallExpression{
									Func:   ast.BareReference{Name: "to_proc"},
									Target: ast.BareReference{Name: "buz"},
								},
							},
						},
					}))
				})
			})

			Context("with args and a block, but no parens", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
Signal.trap "INT", "TERM" do
  MSpec.actions :abort
end
`)
				})

				It("is parsed correctly", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "Signal"},
							Func:   ast.BareReference{Name: "trap"},
							Args: []ast.Node{
								ast.InterpolatedString{Value: "INT"},
								ast.InterpolatedString{Value: "TERM"},
							},
							OptionalBlock: ast.Block{
								Body: []ast.Node{
									ast.CallExpression{
										Target: ast.BareReference{Name: "MSpec"},
										Func:   ast.BareReference{Name: "actions"},
										Args:   []ast.Node{ast.Symbol{Name: "abort"}},
									},
								},
							},
						},
					}))
				})
			})

			Context("with a block passed to a call expression targeting a group", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
    (@repeat || 1).times do
      yield
    end
`)
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.Group{
								Body: []ast.Node{
									ast.CallExpression{
										Target: ast.InstanceVariable{Name: "repeat"},
										Func:   ast.BareReference{Name: "||"},
										Args:   []ast.Node{ast.ConstantInt{Value: 1}},
									},
								},
							},
							Func: ast.BareReference{Name: "times"},
							Args: []ast.Node{},
							OptionalBlock: ast.Block{
								Body: []ast.Node{ast.Yield{}},
							},
						},
					}))
				})
			})

			Context("with args and a block split across newlines", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
method_with_lots_of_args('foo',
                         'bar',
                         'baz') do |foo|
  puts foo
end
`)
				})

				It("should be parsed as though the newlines were not present", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "method_with_lots_of_args"},
							Args: []ast.Node{
								ast.SimpleString{Value: "foo"},
								ast.SimpleString{Value: "bar"},
								ast.SimpleString{Value: "baz"},
							},
							OptionalBlock: ast.Block{
								Args: []ast.Node{ast.BareReference{Name: "foo"}},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "puts"},
										Args: []ast.Node{ast.BareReference{Name: "foo"}},
									},
								},
							},
						},
					}))
				})
			})

			Context("with a proc argument", func() {
				Context("inside parens", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer("takes_a_proc('foo', 'bar', &baz)")
					})

					It("is parsed with an ast.ProcArg argument", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Func: ast.BareReference{Name: "takes_a_proc"},
								Args: []ast.Node{
									ast.SimpleString{Value: "foo"},
									ast.SimpleString{Value: "bar"},
									ast.CallExpression{
										Func:   ast.BareReference{Name: "to_proc"},
										Target: ast.BareReference{Name: "baz"},
									},
								},
							},
						}))
					})
				})

				Context("without parens", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer("takes_a_proc 'foo', 'bar', &baz")
					})

					It("is parsed with an ast.ProcArg argument", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Func: ast.BareReference{Name: "takes_a_proc"},
								Args: []ast.Node{
									ast.SimpleString{Value: "foo"},
									ast.SimpleString{Value: "bar"},
									ast.CallExpression{
										Target: ast.BareReference{Name: "baz"},
										Func:   ast.BareReference{Name: "to_proc"},
									},
								},
							},
						}))
					})
				})
			})

			Context("with a static method", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("foo = String(bar)")
				})

				It("is parsed like a regular method call", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.BareReference{Name: "foo"},
							RHS: ast.CallExpression{
								Func: ast.BareReference{Name: "String"},
								Args: []ast.Node{ast.BareReference{Name: "bar"}},
							},
						},
					}))
				})
			})

			Context("chained together with a block at the end", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
MSpec.retrieve(:files).inject(0) { |max, f| f.size > max ? f.size : max }
`)
				})

				It("is parsed as nested call expressions", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.CallExpression{
								Target: ast.BareReference{Name: "MSpec"},
								Func:   ast.BareReference{Name: "retrieve"},
								Args: []ast.Node{
									ast.Symbol{Name: "files"},
								},
							},
							Func: ast.BareReference{Name: "inject"},
							Args: []ast.Node{
								ast.ConstantInt{Value: 0},
							},
							OptionalBlock: ast.Block{
								Args: []ast.Node{
									ast.BareReference{Name: "max"},
									ast.BareReference{Name: "f"},
								},
								Body: []ast.Node{
									ast.Ternary{
										Condition: ast.CallExpression{
											Target: ast.CallExpression{
												Target: ast.BareReference{Name: "f"},
												Func:   ast.BareReference{Name: "size"},
											},
											Func: ast.BareReference{Name: ">"},
											Args: []ast.Node{ast.BareReference{Name: "max"}},
										},
										True: ast.CallExpression{
											Target: ast.BareReference{Name: "f"},
											Func:   ast.BareReference{Name: "size"},
										},
										False: ast.BareReference{Name: "max"},
									},
								},
							},
						},
					}))
				})
			})

			Context("chained together with dots and parentheses", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("SpecVersion.new(String(other)).to_i")
				})

				It("is parsed as a nested call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.CallExpression{
								Target: ast.BareReference{Name: "SpecVersion"},
								Func:   ast.BareReference{Name: "new"},
								Args: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "String"},
										Args: []ast.Node{ast.BareReference{Name: "other"}},
									},
								},
							},
							Func: ast.BareReference{Name: "to_i"},
						},
					}))
				})
			})

			Context("with an expression wrapped in parentheses", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`('hello %s world' % ['cruel']).inspect`)
				})

				It("is parsed with the expression group as the target", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.Group{
								Body: []ast.Node{
									ast.CallExpression{
										Target: ast.SimpleString{Value: "hello %s world"},
										Func:   ast.BareReference{Name: "%"},
										Args: []ast.Node{
											ast.Array{
												Nodes: []ast.Node{
													ast.SimpleString{Value: "cruel"},
												},
											},
										},
									},
								},
							},
							Func: ast.BareReference{Name: "inspect"},
							Args: []ast.Node{},
						},
					}))
				})
			})

			Context("with methods containing ! and ?", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("5.even?;5.taint!; block_given?; search_and_destroy!")
				})

				It("is parsed just fine", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "even?"},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "taint!"},
						},
						ast.CallExpression{
							Func: ast.BareReference{Name: "block_given?"},
						},
						ast.CallExpression{
							Func: ast.BareReference{Name: "search_and_destroy!"},
						},
					}))
				})
			})

			Context("with a dot", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
5.!
123.abc()
`)
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "!"},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 123},
							Func:   ast.BareReference{Name: "abc"},
							Args:   []ast.Node{},
						},
					}))
				})
			})

			Context("without parens", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("puts 'foo'")
				})

				It("returns a call expression with one arg", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "puts"},
							Args: []ast.Node{ast.SimpleString{Value: "foo"}},
						},
					}))
				})
			})

			Context("with parens", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("puts('foo', 'bar', 'baz')")
				})

				It("returns a call expression with args", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "puts"},
							Args: []ast.Node{
								ast.SimpleString{Value: "foo"},
								ast.SimpleString{Value: "bar"},
								ast.SimpleString{Value: "baz"},
							},
						},
					}))
				})
			})

			Context("without args", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("puts()")
				})

				It("returns a call expression without args", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "puts"},
							Args: []ast.Node{},
						},
					}))
				})
			})

			Context("on an object with args in parentheses", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
foo.send(:catalecta_coassistant)
ARGV.shift
`)
				})

				It("correctly sets the target", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "foo"},
							Func:   ast.BareReference{Name: "send"},
							Args:   []ast.Node{ast.Symbol{Name: "catalecta_coassistant"}},
						},
						ast.CallExpression{
							Target: ast.BareReference{Name: "ARGV"},
							Func:   ast.BareReference{Name: "shift"},
						},
					}))
				})
			})

			Context("a call expression with mixed arguments", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("File.expand_path('../../lib', __FILE__)")
				})

				It("parsed correctly", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "File"},
							Func:   ast.BareReference{Name: "expand_path"},
							Args: []ast.Node{
								ast.SimpleString{Value: "../../lib"},
								ast.FileNameConstReference{},
							},
						},
					}))
				})
			})

			Context("with a call expression as an argument", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("$:.unshift File.expand_path('../../lib', __FILE__)")
				})

				It("applies the correct args to each call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func:   ast.BareReference{Name: "unshift"},
							Target: ast.GlobalVariable{Name: ":"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "File"}, // TODO: should be a class
									Func:   ast.BareReference{Name: "expand_path"},
									Args: []ast.Node{
										ast.SimpleString{Value: "../../lib"},
										ast.FileNameConstReference{},
									},
								},
							},
						},
					}))
				})
			})

			Context("very nested call expressions", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("$:.unshift File.expand_path(File.dirname(__FILE__) + '/../lib')")
				})

				It("is parsed with the correct args on the correct methods", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.GlobalVariable{Name: ":"},
							Func:   ast.BareReference{Name: "unshift"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "File"},
									Func:   ast.BareReference{Name: "expand_path"},
									Args: []ast.Node{
										ast.CallExpression{
											Target: ast.CallExpression{
												Target: ast.BareReference{Name: "File"},
												Func:   ast.BareReference{Name: "dirname"},
												Args:   []ast.Node{ast.FileNameConstReference{}},
											},
											Func: ast.BareReference{Name: "+"},
											Args: []ast.Node{ast.SimpleString{Value: "/../lib"}},
										},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("array splat", func() {
			Context("in a call expression", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("foo(*bar)")
				})

				It("marks the argument for array deferencing", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "foo"},
							Args: []ast.Node{
								ast.StarSplat{Value: ast.BareReference{Name: "bar"}},
							},
						},
					}))
				})
			})
		})

		Describe("method definitions", func() {
			Context("for setter methods", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def foo=(bar)
end

def self.bar=(foo)
end
`)
				})

				It("should be parsed with the equals sign in the name", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Target: nil,
							Name:   ast.BareReference{Name: "foo="},
							Args: []ast.Node{
								ast.MethodParam{
									Name:    ast.BareReference{Name: "bar"},
									IsSplat: false,
								},
							},
							Body: []ast.Node{},
						},
						ast.FuncDecl{
							Target: ast.Self{},
							Name:   ast.BareReference{Name: "bar="},
							Args: []ast.Node{
								ast.MethodParam{
									Name:    ast.BareReference{Name: "foo"},
									IsSplat: false,
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("on an object at runtime", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
obj = Object.new
def obj.start
end
`)
				})

				It("is parsed as a method declaration", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.BareReference{Name: "obj"},
							RHS: ast.CallExpression{
								Target: ast.BareReference{Name: "Object"},
								Func:   ast.BareReference{Name: "new"},
							},
						},
						ast.FuncDecl{
							Target: ast.BareReference{Name: "obj"},
							Name:   ast.BareReference{Name: "start"},
							Body:   []ast.Node{},
							Args:   []ast.Node{},
						},
					}))
				})
			})

			Context("with special names", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def <=>
  self.to_i <=> other
end
`)
				})

				It("parses them as binary operators", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "<=>"},
							Args: []ast.Node{},
							Body: []ast.Node{
								ast.CallExpression{
									Target: ast.CallExpression{
										Target: ast.Self{},
										Func:   ast.BareReference{Name: "to_i"},
									},
									Func: ast.BareReference{Name: "<=>"},
									Args: []ast.Node{ast.BareReference{Name: "other"}},
								},
							},
						},
					}))
				})
			})

			Context("with splat-args", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def on(*args)
end
`)
				})

				It("marks the param as being splatted", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "on"},
							Args: []ast.Node{
								ast.MethodParam{
									Name:    ast.BareReference{Name: "args"},
									IsSplat: true,
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("with a named proc parameter", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def test(*args, &block)
end
`)
				})
				It("marks the param as a proc", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "test"},
							Args: []ast.Node{
								ast.MethodParam{
									Name:    ast.BareReference{Name: "args"},
									IsSplat: true,
								},
								ast.MethodParam{
									Name:   ast.BareReference{Name: "block"},
									IsProc: true,
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("with ? or ! in the name", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def foo!
  raise
end

def foo?
  false
end
`)
				})

				It("is parsed with the correct method name", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "foo!"},
							Args: []ast.Node{},
							Body: []ast.Node{
								ast.BareReference{Name: "raise"},
							},
						},
						ast.FuncDecl{
							Name: ast.BareReference{Name: "foo?"},
							Args: []ast.Node{},
							Body: []ast.Node{
								ast.Boolean{Value: false},
							},
						},
					}))
				})
			})

			Context("without parameters", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def something
  puts 'hai'
end
`)
				})

				It("returns a function declaration with the body set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "something"},
							Args: []ast.Node{},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "hai"}},
								},
							},
						},
					}))
				})
			})

			Context("with parameters with default values", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def foo(a = 123)
end
`)
				})

				It("returns a function declaration with the default values set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{"foo"},
							Args: []ast.Node{
								ast.MethodParam{
									Name:         ast.BareReference{Name: "a"},
									DefaultValue: ast.ConstantInt{Value: 123},
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("with default value args and a block", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def self.describe(mod, options=nil, &block)
end
`)
				})

				It("returns a function declaration with appropriate method parameters", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Target: ast.Self{},
							Name:   ast.BareReference{Name: "describe"},
							Args: []ast.Node{
								ast.MethodParam{Name: ast.BareReference{Name: "mod"}},
								ast.MethodParam{
									Name:         ast.BareReference{Name: "options"},
									DefaultValue: ast.Nil{},
								},
								ast.MethodParam{
									Name:   ast.BareReference{Name: "block"},
									IsProc: true,
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("with parameters surrounded by parens", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def multi_put(str1, str2)
  puts str1
  puts str2
end
`)
				})

				It("returns a function declaration with the args set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "multi_put"},
							Args: []ast.Node{
								ast.MethodParam{Name: ast.BareReference{Name: "str1"}},
								ast.MethodParam{Name: ast.BareReference{Name: "str2"}},
							},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.BareReference{Name: "str1"}},
								},
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.BareReference{Name: "str2"}},
								},
							},
						},
					}))
				})
			})

			Context("with parameters but no parens", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
def multi_put str1, str2
  puts str1, str2
end
`)
				})

				It("returns a function declaration with the args set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.FuncDecl{
							Name: ast.BareReference{Name: "multi_put"},
							Args: []ast.Node{
								ast.MethodParam{Name: ast.BareReference{Name: "str1"}},
								ast.MethodParam{Name: ast.BareReference{Name: "str2"}},
							},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{
										ast.BareReference{Name: "str1"},
										ast.BareReference{Name: "str2"},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("comments", func() {
			Context("on a single line", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("#ceci n'est pas un ligne de code")
				})

				It("is ignored", func() {
					Expect(parser.Statements).To(BeEmpty())
				})
			})

			Context("at the end of a line of code", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
5#this is a comment
12 # this is also a comment
`)
				})

				It("reads the line of code, ignoring the comment", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ConstantInt{Value: 5},
						ast.ConstantInt{Value: 12},
					}))
				})
			})
		})

		Describe("classes", func() {
			Context("without any frills, bells, or whistles", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
class Foo
  puts 'hai'
end
`)
				})

				It("returns a ClassDefn with the body set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ClassDecl{
							Name: "Foo",
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "hai"}},
								},
							},
						},
					}))
				})
			})

			Context("with a superclass", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
class Foo < Bar
end
`)
				})

				It("returns a class with the given superclass", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ClassDecl{
							Name:       "Foo",
							SuperClass: ast.Class{Name: "Bar"},
							Body:       []ast.Node{},
						},
					}))
				})
			})

			Context("with namespaces", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
class Foo::Bar
end
`)
				})

				It("returns a class with the given namespace", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ClassDecl{
							Name:      "Bar",
							Namespace: "Foo",
							Body:      []ast.Node{},
						},
					}))
				})
			})

			Context("with namespaces and super classes", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
class Foo::Biz::Bar < Foo::Biz::Baz
end
`)
				})

				It("returns a class with the given namespace", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ClassDecl{
							Name:       "Bar",
							SuperClass: ast.Class{Name: "Baz", Namespace: "Foo::Biz"},
							Namespace:  "Foo::Biz",
							Body:       []ast.Node{},
						},
					}))
				})
			})

			Context("with a method defined", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
class Whatever < ThisShouldWork
  def initialize
    super

    config[:files] = []
  end
end
`)
				})

				It("should parse a class with a single method defined", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ClassDecl{
							Name:       "Whatever",
							SuperClass: ast.Class{Name: "ThisShouldWork"},
							Body: []ast.Node{
								ast.FuncDecl{
									Name: ast.BareReference{Name: "initialize"},
									Args: []ast.Node{},
									Body: []ast.Node{
										ast.BareReference{Name: "super"},
										ast.CallExpression{
											Target: ast.BareReference{Name: "config"},
											Func:   ast.BareReference{Name: "[]="},
											Args: []ast.Node{
												ast.Symbol{Name: "files"},
												ast.Array{Nodes: []ast.Node{}},
											},
										},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("eigenclasses", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
class Foo
  class << self
    puts 'this is evaluated inside the eigenclass for the Foo class'
  end
end
`)
			})

			It("can be used to modify the unique singleton class for an object", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.ClassDecl{
						Name: "Foo",
						Body: []ast.Node{
							ast.EigenClass{
								Target: ast.Self{},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "puts"},
										Args: []ast.Node{
											ast.SimpleString{
												Value: "this is evaluated inside the eigenclass for the Foo class",
											},
										},
									},
								},
							},
						},
					},
				}))
			})
		})

		// this is ambiguous because it is impossible to determine if you meant
		// foo [:something] as a call expression (e.g.: foo([:an_array_literal]))
		// ... or ...
		// foo [:something] as a call expression on foo (e.g.: call .[] on `foo`)
		Describe("ambiguous [] syntax", func() {
			Context("calling [] and []= on an instance variable", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
@shared [state.to_s] = state
@shared [state.to_s] # returns 'state'
`)
				})

				It("should be parsed as a call expression for the []= operator", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.InstanceVariable{Name: "shared"},
							Func:   ast.BareReference{Name: "[]="},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "state"},
									Func:   ast.BareReference{Name: "to_s"},
								},
								ast.BareReference{Name: "state"},
							},
						},
						ast.CallExpression{
							Target: ast.InstanceVariable{Name: "shared"},
							Func:   ast.BareReference{Name: "[]"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "state"},
									Func:   ast.BareReference{Name: "to_s"},
								},
							},
						},
					}))
				})
			})

			Context("calling [] or []= on a call expression", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("retrieve(:features)[feature] = true")
				})

				It("should be parsed as a call expression for the appropriate method", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.CallExpression{
								Func: ast.BareReference{Name: "retrieve"},
								Args: []ast.Node{ast.Symbol{Name: "features"}},
							},
							Func: ast.BareReference{Name: "[]="},
							Args: []ast.Node{
								ast.BareReference{Name: "feature"},
								ast.Boolean{Value: true},
							},
						},
					}))
				})
			})
		})

		Describe("slicing of an array", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("'1.5.12'.split('.')[0,n]")
			})

			It("should be parsed as a call expression with an array arg", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.CallExpression{
						Target: ast.CallExpression{
							Target: ast.SimpleString{Value: "1.5.12"},
							Func:   ast.BareReference{Name: "split"},
							Args:   []ast.Node{ast.SimpleString{Value: "."}},
						},
						Func: ast.BareReference{Name: "[]"},
						Args: []ast.Node{
							ast.ConstantInt{Value: 0}, ast.BareReference{Name: "n"},
						},
					},
				}))
			})
		})

		Describe("modules", func() {
			Describe("referred to with leading :: to indicate that it belongs to the global namespace", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
SomeModule::InThe::CurrentScope
::SomeModule::InTheGlobalNamespace
::AnotherModule
`)
				})

				It("is parsed with the IsGlobalNamespace field set", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Class{
							Name:              "CurrentScope",
							Namespace:         "SomeModule::InThe",
							IsGlobalNamespace: false,
						},
						ast.Class{
							Name:              "InTheGlobalNamespace",
							Namespace:         "SomeModule",
							IsGlobalNamespace: true,
						},
						ast.Class{
							Name:              "AnotherModule",
							Namespace:         "",
							IsGlobalNamespace: true,
						},
					}))
				})
			})

			Context("with namespaces", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
module Foo::Bar::Baz
puts 'tumescent-wasty'
end
`)
				})

				It("returns a module declaration", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.ModuleDecl{
							Name:      "Baz",
							Namespace: "Foo::Bar",
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "tumescent-wasty"}},
								},
							},
						},
					}))
				})
			})
		})

		Describe("preventing line breaks from terminating an expression", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
# all whitespace is ignored after the / until a nonwhitespace char is seen
foo  \
   # comments here are ignored, as are newlines
   \
   .inspect
`)
			})

			It("can be done using a backslash at the end of a line", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.CallExpression{
						Target: ast.BareReference{Name: "foo"},
						Func:   ast.BareReference{Name: "inspect"},
					},
				}))
			})
		})

		Describe("conditional assignment", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
a ||= 'aftergrass-Dowieite'
@options[:shared] ||= false
`)
			})

			It("returns a ConditionalAssignment expression", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.ConditionalAssignment{
						LHS: ast.BareReference{Name: "a"},
						RHS: ast.SimpleString{Value: "aftergrass-Dowieite"},
					},
					ast.ConditionalAssignment{
						LHS: ast.CallExpression{
							Target: ast.InstanceVariable{Name: "options"},
							Func:   ast.BareReference{Name: "[]"},
							Args:   []ast.Node{ast.Symbol{Name: "shared"}},
						},
						RHS: ast.Boolean{Value: false},
					},
				}))
			})
		})

		Describe("assignment", func() {
			Context("inside of a method call", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("puts(hey = 'so what')")
				})

				It("is parsed as a call expression with assignment as an arg", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "puts"},
							Args: []ast.Node{
								ast.Assignment{
									LHS: ast.BareReference{Name: "hey"},
									RHS: ast.SimpleString{Value: "so what"},
								},
							},
						},
					}))
				})
			})

			Context("to multiple keys in a hash", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
HASH['first_key']    =
HASH['second_key'] = [:something]
`)
				})

				It("is parsed as nested call expressions", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "HASH"},
							Func:   ast.BareReference{Name: "[]="},
							Args: []ast.Node{
								ast.SimpleString{Value: "first_key"},
								ast.CallExpression{
									Target: ast.BareReference{Name: "HASH"},
									Func:   ast.BareReference{Name: "[]="},
									Args: []ast.Node{
										ast.SimpleString{Value: "second_key"},
										ast.Array{
											Nodes: []ast.Node{ast.Symbol{Name: "something"}},
										},
									},
								},
							},
						},
					}))
				})
			})

			Context("to a single variable", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`foo = 5`)
				})

				It("returns an assignment expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.BareReference{Name: "foo"},
							RHS: ast.ConstantInt{Value: 5},
						},
					}))
				})
			})

			Context("to multiple variables", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("foo, bar = [1,2,3]")
				})

				It("returns an assignment expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.Array{
								Nodes: []ast.Node{
									ast.BareReference{Name: "foo"},
									ast.BareReference{Name: "bar"},
								},
							},
							RHS: ast.Array{
								Nodes: []ast.Node{
									ast.ConstantInt{Value: 1},
									ast.ConstantInt{Value: 2},
									ast.ConstantInt{Value: 3},
								},
							},
						},
					}))
				})
			})

			Context("to multiple variables, with a splat", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("target, *actions = clause.split(/([=+-])/)")
				})

				It("is parsed as an assignment expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.Array{Nodes: []ast.Node{
								ast.BareReference{Name: "target"},
								ast.StarSplat{Value: ast.BareReference{Name: "actions"}},
							}},
							RHS: ast.CallExpression{
								Target: ast.BareReference{Name: "clause"},
								Func:   ast.BareReference{Name: "split"},
								Args:   []ast.Node{ast.Regex{Value: "([=+-])"}},
							},
						},
					}))
				})
			})

			Context("to multiple instance or class variables", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
@foo, @bar = [1, 2]
@@foo, @@bar = [3, 4]
`)
				})

				It("should be parsed as a multiple assignment as well", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.Array{
								Nodes: []ast.Node{
									ast.InstanceVariable{Name: "foo"},
									ast.InstanceVariable{Name: "bar"},
								},
							},
							RHS: ast.Array{
								Nodes: []ast.Node{
									ast.ConstantInt{Value: 1},
									ast.ConstantInt{Value: 2},
								},
							},
						},
						ast.Assignment{
							LHS: ast.Array{
								Nodes: []ast.Node{
									ast.ClassVariable{Name: "foo"},
									ast.ClassVariable{Name: "bar"},
								},
							},
							RHS: ast.Array{
								Nodes: []ast.Node{
									ast.ConstantInt{Value: 3},
									ast.ConstantInt{Value: 4},
								},
							},
						},
					}))
				})
			})

			Context("to indices in an array or hash", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("array[i], array[r] = array[r], array[i]")
				})

				It("be parsed as an assignment expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.Array{
								Nodes: []ast.Node{
									ast.CallExpression{
										Target: ast.BareReference{Name: "array"},
										Func:   ast.BareReference{Name: "[]="},
										Args:   []ast.Node{ast.BareReference{Name: "i"}},
									},
									ast.CallExpression{
										Target: ast.BareReference{Name: "array"},
										Func:   ast.BareReference{Name: "[]="},
										Args:   []ast.Node{ast.BareReference{Name: "r"}},
									},
								},
							},
							RHS: ast.Array{
								Nodes: []ast.Node{
									ast.CallExpression{
										Target: ast.BareReference{Name: "array"},
										Func:   ast.BareReference{Name: "[]="},
										Args:   []ast.Node{ast.BareReference{Name: "r"}},
									},
									ast.CallExpression{
										Target: ast.BareReference{Name: "array"},
										Func:   ast.BareReference{Name: "[]="},
										Args:   []ast.Node{ast.BareReference{Name: "i"}},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("booleans", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
true
false
`)
			})

			It("returns a Boolean", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Boolean{Value: true},
					ast.Boolean{Value: false},
				}))
			})
		})

		Describe("unary operators", func() {
			Describe("unary NOT", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`!!true`)
				})

				It("returns a Negation expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Negation{
							ast.Negation{
								Target: ast.Boolean{Value: true},
							},
						},
					}))
				})
			})

			Describe("unary COMPLEMENT", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("~~false")
				})

				It("returns a Complement expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Complement{
							ast.Complement{
								Target: ast.Boolean{Value: false},
							},
						},
					}))
				})
			})

			Describe("unary plus", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("++foo")
				})

				It("returns a Positive expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Positive{
							ast.Positive{
								Target: ast.BareReference{Name: "foo"},
							},
						},
					}))
				})
			})

			Describe("unary minus", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("--867.5309")
				})

				It("returns a Negative expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Negative{
							ast.Negative{
								Target: ast.ConstantFloat{Value: 867.5309},
							},
						},
					}))
				})
			})
		})

		Describe("binary operators", func() {
			Describe("+", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("5 + 12")
				})

				It("returns a CallExpression with the '+' method", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "+"},
							Args:   []ast.Node{ast.ConstantInt{Value: 12}},
						},
					}))
				})
			})

			Describe("-", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("555 - 123")
				})

				It("returns a subtraction call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 555},
							Func:   ast.BareReference{Name: "-"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
					}))
				})
			})

			Describe("*", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("321 * 123")
				})

				It("returns a multiplication call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 321},
							Func:   ast.BareReference{Name: "*"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
					}))
				})
			})

			Describe("/", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
321 / 123
(abc) / (xyz)
`)
				})

				It("returns a division call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 321},
							Func:   ast.BareReference{Name: "/"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
						ast.CallExpression{
							Target: ast.Group{Body: []ast.Node{ast.BareReference{Name: "abc"}}},
							Func:   ast.BareReference{Name: "/"},
							Args:   []ast.Node{ast.Group{Body: []ast.Node{ast.BareReference{Name: "xyz"}}}},
						},
					}))
				})
			})

			Describe("%", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("321 % 123")
				})

				It("returns a modulo call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 321},
							Func:   ast.BareReference{Name: "%"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
					}))
				})
			})

			Describe("**", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("321 ** 123")
				})

				It("returns a modulo call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 321},
							Func:   ast.BareReference{Name: "**"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
					}))
				})
			})

			Describe("<< and >>", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
321 << 123
555 >> 666
`)
				})

				It("returns a shovel call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 321},
							Func:   ast.BareReference{Name: "<<"},
							Args:   []ast.Node{ast.ConstantInt{Value: 123}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 555},
							Func:   ast.BareReference{Name: ">>"},
							Args:   []ast.Node{ast.ConstantInt{Value: 666}},
						},
					}))
				})
			})

			Describe("bitwise binary operators", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
1 & 0
1 | 0
1 ^ 5
File.lchmod mode & 01777, path
`)
				})

				It("returns a call expressions for those methods", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "&"},
							Args:   []ast.Node{ast.ConstantInt{Value: 0}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "|"},
							Args:   []ast.Node{ast.ConstantInt{Value: 0}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "^"},
							Args:   []ast.Node{ast.ConstantInt{Value: 5}},
						},
						ast.CallExpression{
							Target: ast.BareReference{Name: "File"},
							Func:   ast.BareReference{Name: "lchmod"},
							Args: []ast.Node{
								ast.CallExpression{
									Target: ast.BareReference{Name: "mode"},
									Func:   ast.BareReference{Name: "&"},
									Args:   []ast.Node{ast.ConstantInt{Value: 1777}},
								},
								ast.BareReference{Name: "path"},
							},
						},
					}))
				})
			})

			Describe("operators for sorting", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
0 < 1
1 > 0
5 <= 55
12 >= 21
22 <=> 22
`)
				})

				It("is parsed as an call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 0},
							Func:   ast.BareReference{Name: "<"},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: ">"},
							Args:   []ast.Node{ast.ConstantInt{Value: 0}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "<="},
							Args:   []ast.Node{ast.ConstantInt{Value: 55}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 12},
							Func:   ast.BareReference{Name: ">="},
							Args:   []ast.Node{ast.ConstantInt{Value: 21}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 22},
							Func:   ast.BareReference{Name: "<=>"},
							Args:   []ast.Node{ast.ConstantInt{Value: 22}},
						},
					}))
				})
			})

			Describe("binary boolean operators", func() {
				Context("with simple types on the left and right side", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
1 && 0
1 || 0
1 and 0
1 or 0
`)
					})

					// FIXME: these first two should NOT be call expressions
					It("is parsed as a call expression", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Target: ast.ConstantInt{Value: 1},
								Func:   ast.BareReference{Name: "&&"},
								Args:   []ast.Node{ast.ConstantInt{Value: 0}},
							},
							ast.CallExpression{
								Target: ast.ConstantInt{Value: 1},
								Func:   ast.BareReference{Name: "||"},
								Args:   []ast.Node{ast.ConstantInt{Value: 0}},
							},
							ast.WeakLogicalAnd{
								LHS: ast.ConstantInt{Value: 1},
								RHS: ast.ConstantInt{Value: 0},
							},
							ast.WeakLogicalOr{
								LHS: ast.ConstantInt{Value: 1},
								RHS: ast.ConstantInt{Value: 0},
							},
						}))
					})
				})

				Context("with complex types on the left and right side", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`retrieve(:features)[feature] || false`)
					})

					It("is parsed as a call expression", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.CallExpression{
								Target: ast.CallExpression{
									Target: ast.CallExpression{
										Func: ast.BareReference{Name: "retrieve"},
										Args: []ast.Node{ast.Symbol{Name: "features"}},
									},
									Func: ast.BareReference{Name: "[]"},
									Args: []ast.Node{ast.BareReference{Name: "feature"}},
								},
								Func: ast.BareReference{Name: "||"},
								Args: []ast.Node{ast.Boolean{Value: false}},
							},
						}))
					})
				})
			})

			Describe("equality operators", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
1 == 1
1 === 1
1 != 1
1 =~ 1
1 !~ 1
`)
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "=="},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "==="},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "!="},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "=~"},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 1},
							Func:   ast.BareReference{Name: "!~"},
							Args:   []ast.Node{ast.ConstantInt{Value: 1}},
						},
					}))
				})
			})
		})

		Describe("arrays", func() {
			Context("with newlines between elements", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`[1,2,3,
4,
5,
6
]`)
				})

				It("is parsed as an Array", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Array{Nodes: []ast.Node{
							ast.ConstantInt{Value: 1},
							ast.ConstantInt{Value: 2},
							ast.ConstantInt{Value: 3},
							ast.ConstantInt{Value: 4},
							ast.ConstantInt{Value: 5},
							ast.ConstantInt{Value: 6},
						}},
					}))
				})
			})

			Context("as the target and argument of a binary operator", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("[1,2,3,4,5] - [1]")
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.Array{Nodes: []ast.Node{
								ast.ConstantInt{Value: 1},
								ast.ConstantInt{Value: 2},
								ast.ConstantInt{Value: 3},
								ast.ConstantInt{Value: 4},
								ast.ConstantInt{Value: 5},
							}},
							Func: ast.BareReference{Name: "-"},
							Args: []ast.Node{
								ast.Array{Nodes: []ast.Node{
									ast.ConstantInt{Value: 1},
								}},
							},
						},
					}))
				})
			})

			Context("of simple built-in types", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("[1,2,3,   4,5,6 ]")
				})

				It("is parsed as an Array", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Array{
							Nodes: []ast.Node{
								ast.ConstantInt{Value: 1},
								ast.ConstantInt{Value: 2},
								ast.ConstantInt{Value: 3},
								ast.ConstantInt{Value: 4},
								ast.ConstantInt{Value: 5},
								ast.ConstantInt{Value: 6},
							},
						},
					}))
				})
			})

			Context("of named variables", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("[cladosiphonic, capillitial, bicarbureted, argentose]")
				})

				It("is parsed as an Array", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Array{
							Nodes: []ast.Node{
								ast.BareReference{Name: "cladosiphonic"},
								ast.BareReference{Name: "capillitial"},
								ast.BareReference{Name: "bicarbureted"},
								ast.BareReference{Name: "argentose"},
							},
						},
					}))
				})
			})

			Context("of arrays", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("[[], [], []]")
				})

				It("is parsed as an Array", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Array{Nodes: []ast.Node{
							ast.Array{Nodes: []ast.Node{}},
							ast.Array{Nodes: []ast.Node{}},
							ast.Array{Nodes: []ast.Node{}},
						}},
					}))
				})
			})
		})

		Describe("hashes", func() {
			Context("with each key-value pair on a newline and no comma on the last line", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
# these two hashes are the same
{
  :foo => :bar,
  :bar => :foo
}

{
  :foo => :bar,
  :bar => :foo,
}

`)
				})

				It("returns a Hash node", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Hash{
							Pairs: []ast.HashKeyValuePair{
								{
									Key:   ast.Symbol{Name: "foo"},
									Value: ast.Symbol{Name: "bar"},
								},
								{
									Key:   ast.Symbol{Name: "bar"},
									Value: ast.Symbol{Name: "foo"},
								},
							},
						},
						ast.Hash{
							Pairs: []ast.HashKeyValuePair{
								{
									Key:   ast.Symbol{Name: "foo"},
									Value: ast.Symbol{Name: "bar"},
								},
								{
									Key:   ast.Symbol{Name: "bar"},
									Value: ast.Symbol{Name: "foo"},
								},
							},
						},
					}))
				})
			})

			Context("without anything inside", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("{}")
				})

				It("is parsed as a hash", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Hash{},
					}))
				})
			})

			Context("with hashrockets", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("{:foo => bar}")
				})

				It("returns a Hash node", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Hash{
							Pairs: []ast.HashKeyValuePair{
								{
									Key:   ast.Symbol{Name: "foo"},
									Value: ast.BareReference{Name: "bar"},
								},
							},
						},
					}))
				})
			})

			Describe("assigning a value via a setter", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("Sharpware.nasality = 'cladosiphonic-capillitial'")
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "Sharpware"},
							Func:   ast.BareReference{Name: "nasality="},
							Args: []ast.Node{
								ast.SimpleString{Value: "cladosiphonic-capillitial"},
							},
						},
					}))
				})
			})

			Describe("assigning a value to a key", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("hash[:key] = :value")
				})

				It("is parsed as a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "hash"},
							Func:   ast.BareReference{Name: "[]="},
							Args: []ast.Node{
								ast.Symbol{Name: "key"},
								ast.Symbol{Name: "value"},
							},
						},
					}))
				})
			})

			Describe("retrieving a value for a given key", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("hash[:key]")
				})

				It("returns a call expression", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "hash"},
							Func:   ast.BareReference{Name: "[]"},
							Args:   []ast.Node{ast.Symbol{Name: "key"}},
						},
					}))
				})
			})

			Context("with 1.9 'key: value' pairs", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`{
key: value,
foo: bar,
}`)
				})

				It("returns a Hash node with symbol keys", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Hash{
							Pairs: []ast.HashKeyValuePair{
								{
									Key:   ast.Symbol{Name: "key"},
									Value: ast.BareReference{Name: "value"},
								}, {
									Key:   ast.Symbol{Name: "foo"},
									Value: ast.BareReference{Name: "bar"},
								},
							},
						},
					}))
				})
			})
		})

		Describe("globals", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("$LOAD_PATH; $0")
			})

			It("should be parsed as a GlobalVariable", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.GlobalVariable{Name: "LOAD_PATH"},
					ast.GlobalVariable{Name: "0"},
				}))
			})
		})

		Describe("instance variables", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
@foo = :bar
@FOO = :baz
`)
			})

			It("should be parsed as an InstanceVariable", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Assignment{
						LHS: ast.InstanceVariable{Name: "foo"},
						RHS: ast.Symbol{Name: "bar"},
					},
					ast.Assignment{
						LHS: ast.InstanceVariable{Name: "FOO"},
						RHS: ast.Symbol{Name: "baz"},
					},
				}))
			})
		})

		Describe("class variables", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
@@foo = :bar
@@FOO = :baz
`)
			})

			It("should be parsed as an ClassVariable", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Assignment{
						LHS: ast.ClassVariable{Name: "foo"},
						RHS: ast.Symbol{Name: "bar"},
					},
					ast.Assignment{
						LHS: ast.ClassVariable{Name: "FOO"},
						RHS: ast.Symbol{Name: "baz"},
					},
				}))
			})
		})

		Describe("lambdas", func() {
			Context("without any frills", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("something = lambda { puts 'hai'; exit }")
				})

				It("is parsed as an ast.Lambda", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.BareReference{Name: "something"},
							RHS: ast.Lambda{
								Body: ast.Block{
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "hai"}},
										},
										ast.BareReference{Name: "exit"},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("a method with rescue statements at the end", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
def samsonic_obey
  raise 'whoopsie-daisy'
rescue Nope
  puts 'unlikely'
rescue NotThisEither => e
  puts 'contrived'
rescue Errno::ENOTEMPTY, Errno::ENOENT
  puts 'less contrived'
rescue
  puts 'aww yisss'
end
`)
			})

			It("is parsed as rescue statements", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.FuncDecl{
						Name: ast.BareReference{Name: "samsonic_obey"},
						Args: []ast.Node{},
						Body: []ast.Node{
							ast.CallExpression{
								Func: ast.BareReference{Name: "raise"},
								Args: []ast.Node{ast.SimpleString{Value: "whoopsie-daisy"}},
							},
						},
						Rescues: []ast.Node{
							ast.Rescue{
								Exception: ast.RescueException{
									Classes: []ast.Class{{Name: "Nope"}},
								},
								Body: []ast.Node{ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "unlikely"}},
								}},
							},
							ast.Rescue{
								Exception: ast.RescueException{
									Classes: []ast.Class{{Name: "NotThisEither"}},
									Var:     ast.BareReference{Name: "e"},
								},
								Body: []ast.Node{ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "contrived"}},
								}},
							},
							ast.Rescue{
								Exception: ast.RescueException{
									Classes: []ast.Class{
										{
											Name:      "ENOTEMPTY",
											Namespace: "Errno",
										}, {
											Name:      "ENOENT",
											Namespace: "Errno",
										},
									},
								},
								Body: []ast.Node{ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "less contrived"}},
								}},
							},
							ast.Rescue{
								Exception: ast.RescueException{},
								Body: []ast.Node{ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "aww yisss"}},
								}},
							},
						},
					},
				}))
			})
		})

		Describe("the 'alias' keyword", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
module Foo
  alias wat zomg
  alias seriously? yes_srsly!
end
`)
			})

			It("does not allow commas and converts its 'args' to symbols", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.ModuleDecl{
						Name: "Foo",
						Body: []ast.Node{
							ast.Alias{
								To:   ast.Symbol{Name: "wat"},
								From: ast.Symbol{Name: "zomg"},
							},
							ast.Alias{
								To:   ast.Symbol{Name: "seriously?"},
								From: ast.Symbol{Name: "yes_srsly!"},
							},
						},
					},
				}))
			})
		})

		Describe("the retry keyword", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
begin
  retry if falsey_method()
  retry
end
`)
			})

			It("should be parsed as a Retry node", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Begin{
						Body: []ast.Node{
							ast.IfBlock{
								Condition: ast.CallExpression{
									Args: []ast.Node{},
									Func: ast.BareReference{Name: "falsey_method"},
								},
								Body: []ast.Node{ast.Retry{}},
							},
							ast.Retry{},
						},
						Rescue: []ast.Node{},
					},
				}))
			})
		})

		Describe("with conditional returns inside a method", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
def method_with_conditional_return
  names.each do |name|
    return Kernel.load(name) if File.exist?(File.expand_path(name))
    return unless false
  end
end
`)
			})

			It("should be parsed correctly", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.FuncDecl{
						Name: ast.BareReference{Name: "method_with_conditional_return"},
						Args: []ast.Node{},
						Body: []ast.Node{
							ast.CallExpression{
								Target: ast.BareReference{Name: "names"},
								Func:   ast.BareReference{Name: "each"},
								Args:   []ast.Node{},
								OptionalBlock: ast.Block{
									Args: []ast.Node{ast.BareReference{Name: "name"}},
									Body: []ast.Node{
										ast.IfBlock{
											Condition: ast.CallExpression{
												Target: ast.BareReference{Name: "File"},
												Func:   ast.BareReference{Name: "exist?"},
												Args: []ast.Node{
													ast.CallExpression{
														Target: ast.BareReference{Name: "File"},
														Func:   ast.BareReference{Name: "expand_path"},
														Args:   []ast.Node{ast.BareReference{Name: "name"}},
													},
												},
											},
											Body: []ast.Node{
												ast.Return{
													Value: ast.CallExpression{
														Target: ast.BareReference{Name: "Kernel"},
														Func:   ast.BareReference{Name: "load"},
														Args:   []ast.Node{ast.BareReference{Name: "name"}},
													},
												},
											},
										},
										ast.IfBlock{
											Condition: ast.Negation{Target: ast.Boolean{Value: false}},
											Body: []ast.Node{
												ast.Return{},
											},
										},
									},
								},
							},
						},
					},
				}))
			})
		})

		Describe("with expressions that return a value", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
def with_a_block()
  yield 'nonrestricted-consonantize'
  return 'albitite-compotor', 'catalecta-coassistant', 'semiannual-pomfret'
end
`)
			})

			It("should parse the expressions correctly", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.FuncDecl{
						Name: ast.BareReference{Name: "with_a_block"},
						Args: []ast.Node{},
						Body: []ast.Node{
							ast.Yield{
								Value: ast.SimpleString{
									Value: "nonrestricted-consonantize",
								},
							},
							ast.Return{
								Value: ast.Nodes{
									ast.SimpleString{Value: "albitite-compotor"},
									ast.SimpleString{Value: "catalecta-coassistant"},
									ast.SimpleString{Value: "semiannual-pomfret"},
								},
							},
						},
					},
				}))
			})
		})

		Describe("blocks", func() {
			Context("without any args", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
function_that_takes_a_block do
  puts 'semiannual-pomfret'
end
`)
				})

				It("is parsed as an ast.Block", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "function_that_takes_a_block"},
							Args: []ast.Node{},
							OptionalBlock: ast.Block{
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "puts"},
										Args: []ast.Node{ast.SimpleString{Value: "semiannual-pomfret"}},
									},
								},
							},
						},
					}))
				})
			})

			Context("with args", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
with_a_block do |with, some, args|
  'aww yiss'
end
`)
				})

				It("is parsed as an ast.Block with args", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Func: ast.BareReference{Name: "with_a_block"},
							Args: []ast.Node{},
							OptionalBlock: ast.Block{
								Args: []ast.Node{
									ast.BareReference{Name: "with"},
									ast.BareReference{Name: "some"},
									ast.BareReference{Name: "args"},
								},
								Body: []ast.Node{ast.SimpleString{Value: "aww yiss"}},
							},
						},
					}))
				})
			})

			Context("with curly braces", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("with.a_block {|foo| puts foo}")
				})

				It("is parsed as an ast.Block with args", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.BareReference{Name: "with"},
							Func:   ast.BareReference{Name: "a_block"},
							Args:   []ast.Node{},
							OptionalBlock: ast.Block{
								Args: []ast.Node{ast.BareReference{Name: "foo"}},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "puts"},
										Args: []ast.Node{ast.BareReference{Name: "foo"}},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("ranges", func() {
			Context("of integers", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("-1..-5")
				})

				It("should be parsed as a Range", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Range{
							Start: ast.Negative{Target: ast.ConstantInt{Value: 1}},
							End:   ast.Negative{Target: ast.ConstantInt{Value: 5}},
						},
					}))
				})
			})
		})

		Describe("% notation", func() {
			Describe("for regular expressions", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("%r(string/)")
				})

				It("should parse it as an ast.Regex", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Regex{Value: "string/"},
					}))
				})
			})
		})

		Describe("regex literals", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("/^foo.*bar$/")
			})

			It("is parsed as an ast.Regex", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Regex{Value: "^foo.*bar$"},
				}))
			})
		})

		Describe("unless", func() {
			Context("at the end of an expression", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("5 unless false")
				})

				It("is parsed as an IfBlock", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Negation{Target: ast.Boolean{false}},
							Body:      []ast.Node{ast.ConstantInt{Value: 5}},
						},
					}))
				})
			})

			Context("with a comparision of call expressions", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
unless target[0..1] == config[:config_ext]
end
`)
				})

				It("is parsed as an IfBlock", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Negation{
								ast.CallExpression{
									Target: ast.CallExpression{
										Target: ast.BareReference{Name: "target"},
										Func:   ast.BareReference{Name: "[]"},
										Args: []ast.Node{
											ast.Range{
												Start: ast.ConstantInt{Value: 0},
												End:   ast.ConstantInt{Value: 1},
											},
										},
									},
									Func: ast.BareReference{Name: "=="},
									Args: []ast.Node{
										ast.CallExpression{
											Target: ast.BareReference{Name: "config"},
											Func:   ast.BareReference{Name: "[]"},
											Args:   []ast.Node{ast.Symbol{Name: "config_ext"}},
										},
									},
								},
							},
							Body: []ast.Node{},
						},
					}))
				})
			})

			Context("inside another unless", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
unless @zomg
  no_wai = 9999 unless yes_wai
  yes_wai = 9999 unless no_wai
end
`)
				})

				It("is parsed as a nested IfBlock", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Negation{Target: ast.InstanceVariable{Name: "zomg"}},
							Body: []ast.Node{
								ast.IfBlock{
									Condition: ast.Negation{Target: ast.BareReference{Name: "yes_wai"}},
									Body: []ast.Node{
										ast.Assignment{
											LHS: ast.BareReference{Name: "no_wai"},
											RHS: ast.ConstantInt{Value: 9999},
										},
									},
								},
								ast.IfBlock{
									Condition: ast.Negation{Target: ast.BareReference{Name: "no_wai"}},
									Body: []ast.Node{
										ast.Assignment{
											LHS: ast.BareReference{Name: "yes_wai"},
											RHS: ast.ConstantInt{Value: 9999},
										},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("if else blocks", func() {
			Context("without an else", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
if false
  puts 'Romanize-whereover'
end`)
				})

				It("is parsed as an ast.IfBlock struct", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Boolean{Value: false},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "Romanize-whereover"}},
								},
							},
						},
					}))
				})
			})

			Context("at the end of an expression", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
exit(1) if false
foo = :bar if something.truthy_method
raise OptionError, "description" if args.size < 2
`)
				})

				It("is parsed as an ast.IfBlock", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Boolean{Value: false},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "exit"},
									Args: []ast.Node{ast.ConstantInt{Value: 1}},
								},
							},
						},
						ast.IfBlock{
							Condition: ast.CallExpression{
								Target: ast.BareReference{Name: "something"},
								Func:   ast.BareReference{Name: "truthy_method"},
							},
							Body: []ast.Node{
								ast.Assignment{
									LHS: ast.BareReference{Name: "foo"},
									RHS: ast.Symbol{Name: "bar"},
								},
							},
						},
						ast.IfBlock{
							Condition: ast.CallExpression{
								Target: ast.CallExpression{
									Target: ast.BareReference{Name: "args"},
									Func:   ast.BareReference{Name: "size"},
								},
								Func: ast.BareReference{Name: "<"},
								Args: []ast.Node{ast.ConstantInt{Value: 2}},
							},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "raise"},
									Args: []ast.Node{
										ast.BareReference{Name: "OptionError"},
										ast.InterpolatedString{Value: "description"},
									},
								},
							},
						},
					}))
				})
			})

			Context("with an else", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
if false
  puts 'Romanize-whereover'
else
  puts 'Kiplingese-disinvolve'
end`)
				})

				It("is parsed as an ast.IfBlock struct", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Boolean{Value: false},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "Romanize-whereover"}},
								},
							},
							Else: []ast.Node{
								ast.IfBlock{
									Condition: ast.Boolean{Value: true},
									Body: []ast.Node{
										ast.CallExpression{
											Func: ast.BareReference{Name: "puts"},
											Args: []ast.Node{ast.SimpleString{Value: "Kiplingese-disinvolve"}},
										},
									},
								},
							},
						},
					}))
				})
			})

			Context("with multiple elsif conditions", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
if false
  'purifier-cartouche'
elsif false
  'bronchophthisis-hypersurface'
elsif false
  'sharpware-nasality'
else
  'Osmeridae-harpylike'
end
`)
				})

				It("returns a very nested set of conditions to check", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.IfBlock{
							Condition: ast.Boolean{Value: false},
							Body: []ast.Node{
								ast.SimpleString{Value: "purifier-cartouche"},
							},
							Else: []ast.Node{
								ast.IfBlock{
									Condition: ast.Boolean{Value: false},
									Body: []ast.Node{
										ast.SimpleString{Value: "bronchophthisis-hypersurface"},
									},
								},
								ast.IfBlock{
									Condition: ast.Boolean{Value: false},
									Body: []ast.Node{
										ast.SimpleString{Value: "sharpware-nasality"},
									},
								},
								ast.IfBlock{
									Condition: ast.Boolean{Value: true},
									Body: []ast.Node{
										ast.SimpleString{Value: "Osmeridae-harpylike"},
									},
								},
							},
						},
					}))
				})
			})
		})

		Describe("an expression that can fail followed by rescue", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("value = can_raise() rescue 'whoops'")
			})

			It("uses the provided value as a fall back for assignment", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Assignment{
						LHS: ast.BareReference{Name: "value"},
						RHS: ast.RescueModifier{
							Statement: ast.CallExpression{
								Args: []ast.Node{},
								Func: ast.BareReference{Name: "can_raise"},
							},
							Rescue: ast.SimpleString{Value: "whoops"},
						},
					},
				}))
			})
		})

		Describe("rescuing without a class, and capturing the exception thrown", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
begin
rescue => wat
end
`)
			})

			It("should be parsed as a BeginBlock struct", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Begin{
						Body: []ast.Node{},
						Rescue: []ast.Node{
							ast.Rescue{
								Body: []ast.Node{},
								Exception: ast.RescueException{
									Var: ast.BareReference{Name: "wat"},
								},
							},
						},
					},
				}))
			})
		})

		Describe("an else clause for begin / rescue / else / end", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
begin
  'this always happens'
rescue
  'only when an exception occurs in the begin scope'
else
  'this only happens if there were no exceptions'
end
`)
			})

			It("should be parsed as a BeginBlock struct", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Begin{
						Body: []ast.Node{ast.SimpleString{Value: "this always happens"}},
						Rescue: []ast.Node{
							ast.Rescue{Body: []ast.Node{
								ast.SimpleString{
									Value: "only when an exception occurs in the begin scope",
								},
							}},
						},
						Else: []ast.Node{
							ast.SimpleString{Value: "this only happens if there were no exceptions"},
						},
					},
				}))
			})
		})

		Describe("begin followed by several rescue statements", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
begin
  foo()
rescue
  bar()
rescue LoadError
  biz()
rescue Exception => e
  baz()
end
`)
			})

			It("is parsed as a BeginBlock struct", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Begin{
						Body: []ast.Node{
							ast.CallExpression{
								Func: ast.BareReference{Name: "foo"},
								Args: []ast.Node{},
							},
						},
						Rescue: []ast.Node{
							ast.Rescue{
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "bar"},
										Args: []ast.Node{},
									},
								},
							},
							ast.Rescue{
								Exception: ast.RescueException{
									Classes: []ast.Class{{Name: "LoadError"}},
								},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "biz"},
										Args: []ast.Node{},
									},
								},
							},
							ast.Rescue{
								Exception: ast.RescueException{
									Var:     ast.BareReference{Name: "e"},
									Classes: []ast.Class{{Name: "Exception"}},
								},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "baz"},
										Args: []ast.Node{},
									},
								},
							},
						},
					},
				}))
			})
		})

		Describe("ternary ?", func() {
			Context("as the right hand side of an assignment expression", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
s = short ? short.dup : "  "
`)
				})

				It("is parsed correctly", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Assignment{
							LHS: ast.BareReference{Name: "s"},
							RHS: ast.Ternary{
								Condition: ast.BareReference{Name: "short"},
								True: ast.CallExpression{
									Target: ast.BareReference{Name: "short"},
									Func:   ast.BareReference{Name: "dup"},
								},
								False: ast.InterpolatedString{Value: "  "},
							},
						},
					}))
				})
			})

			Context("with simple values", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("true ? 'string' : 5")
				})

				It("should be parsed as a Ternary node", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Ternary{
							Condition: ast.Boolean{Value: true},
							True:      ast.SimpleString{Value: "string"},
							False:     ast.ConstantInt{Value: 5},
						},
					}))
				})
			})

			Context("with call expressions", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer("what() ? mm.hmm : thats_cool()")
				})

				It("should be parsed as a Ternary node", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Ternary{
							Condition: ast.CallExpression{
								Func: ast.BareReference{Name: "what"},
								Args: []ast.Node{},
							},
							True: ast.CallExpression{
								Target: ast.BareReference{Name: "mm"},
								Func:   ast.BareReference{Name: "hmm"},
							},
							False: ast.CallExpression{
								Func: ast.BareReference{Name: "thats_cool"},
								Args: []ast.Node{},
							},
						},
					}))
				})
			})
		})

		Describe("semicolons", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(";;a; b; c;;")
			})

			It("are treated as a line separator", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.BareReference{Name: "a"},
					ast.BareReference{Name: "b"},
					ast.BareReference{Name: "c"},
				}))
			})
		})

		Describe("parentheses", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
(1)
('hey'; 'this'; 'works!')
([].unshift)
('hello %s world' % 'grubby')
`)
			})

			It("is treated as a grouping for a set of expressions", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Group{
						Body: []ast.Node{ast.ConstantInt{Value: 1}},
					},
					ast.Group{
						Body: []ast.Node{
							ast.SimpleString{Value: "hey"},
							ast.SimpleString{Value: "this"},
							ast.SimpleString{Value: "works!"},
						},
					},
					ast.Group{
						Body: []ast.Node{
							ast.CallExpression{
								Target: ast.Array{Nodes: []ast.Node{}},
								Func:   ast.BareReference{Name: "unshift"},
							},
						},
					},
					ast.Group{
						Body: []ast.Node{
							ast.CallExpression{
								Target: ast.SimpleString{Value: "hello %s world"},
								Func:   ast.BareReference{Name: "%"},
								Args:   []ast.Node{ast.SimpleString{Value: "grubby"}},
							},
						},
					},
				}))
			})
		})

		Describe("modules", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
Foo::Bar
Foo::Bar::Baz.method_call
`)
			})

			It("can be used to refer to classes and methods inside modules", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.Class{
						Name:      "Bar",
						Namespace: "Foo",
					},
					ast.CallExpression{
						Target: ast.Class{
							Name:      "Baz",
							Namespace: "Foo::Bar",
						},
						Func: ast.BareReference{Name: "method_call"},
					},
				}))
			})
		})

		Describe("memoizing methods", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
def memoized_func
  unless @value
    @value = expensive_function_call()
  end

  @value
end
`)
			})

			It("should be parsed correctly", func() {
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.FuncDecl{
						Name: ast.BareReference{Name: "memoized_func"},
						Args: []ast.Node{},
						Body: []ast.Node{
							ast.IfBlock{
								Condition: ast.Negation{ast.InstanceVariable{Name: "value"}},
								Body: []ast.Node{
									ast.Assignment{
										LHS: ast.InstanceVariable{Name: "value"},
										RHS: ast.CallExpression{
											Func: ast.BareReference{Name: "expensive_function_call"},
											Args: []ast.Node{},
										},
									},
								},
							},
							ast.InstanceVariable{Name: "value"},
						},
					},
				}))
			})
		})

		Describe("loops", func() {
			Context("with blocks", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
5.times do
  next
end
`)
				})

				It("is parsed as a block, but the 'next' keyword is valid here", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.CallExpression{
							Target: ast.ConstantInt{Value: 5},
							Func:   ast.BareReference{Name: "times"},
							Args:   []ast.Node{},
							OptionalBlock: ast.Block{
								Body: []ast.Node{ast.Next{}},
							},
						},
					}))
				})
			})

			Context("with an until statement", func() {
				BeforeEach(func() {
					lexer = parser.NewLexer(`
until 1 == 2
  puts 'INFINITE LOOP AHOY!!!!1'
end
`)
				})

				It("is parsed as a Loop", func() {
					Expect(parser.Statements).To(Equal([]ast.Node{
						ast.Loop{
							Condition: ast.Negation{
								Target: ast.CallExpression{
									Target: ast.ConstantInt{Value: 1},
									Func:   ast.BareReference{Name: "=="},
									Args:   []ast.Node{ast.ConstantInt{Value: 2}},
								},
							},
							Body: []ast.Node{
								ast.CallExpression{
									Func: ast.BareReference{Name: "puts"},
									Args: []ast.Node{ast.SimpleString{Value: "INFINITE LOOP AHOY!!!!1"}},
								},
							},
						},
					}))
				})
			})

			Describe("while statements", func() {
				Context("at the end of an expression", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
'check this out' while false

begin
  puts 'whaaat'
end while true
`)
					})

					It("are valid loops too", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.Loop{
								Condition: ast.Boolean{Value: false},
								Body: []ast.Node{
									ast.SimpleString{Value: "check this out"},
								},
							},
							ast.Loop{
								Condition: ast.Boolean{Value: true},
								Body: []ast.Node{
									ast.Begin{
										Body: []ast.Node{
											ast.CallExpression{
												Func: ast.BareReference{Name: "puts"},
												Args: []ast.Node{ast.SimpleString{Value: "whaaat"}},
											},
										},
										Rescue: []ast.Node{},
									},
								},
							},
						}))
					})
				})

				Context("with a trailing end keyword", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
while foo = bar.baz
  puts 'welp'
  break if false
  next if false
end
`)
					})

					It("is parsed into a Loop struct", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.Loop{
								Condition: ast.Assignment{
									LHS: ast.BareReference{Name: "foo"},
									RHS: ast.CallExpression{
										Target: ast.BareReference{Name: "bar"},
										Func:   ast.BareReference{Name: "baz"},
									},
								},
								Body: []ast.Node{
									ast.CallExpression{
										Func: ast.BareReference{Name: "puts"},
										Args: []ast.Node{ast.SimpleString{Value: "welp"}},
									},
									ast.IfBlock{
										Condition: ast.Boolean{Value: false},
										Body:      []ast.Node{ast.Break{}},
									},
									ast.IfBlock{
										Condition: ast.Boolean{Value: false},
										Body:      []ast.Node{ast.Next{}},
									},
								},
							},
						}))
					})
				})

				Context("with a deeply nested next keyword", func() {
					BeforeEach(func() {
						lexer = parser.NewLexer(`
while true
  if false
    if false
      next
    end
  end
end
`)
					})

					It("parses correctly", func() {
						Expect(parser.Statements).To(Equal([]ast.Node{
							ast.Loop{
								Condition: ast.Boolean{Value: true},
								Body: []ast.Node{
									ast.IfBlock{
										Condition: ast.Boolean{Value: false},
										Body: []ast.Node{
											ast.IfBlock{
												Condition: ast.Boolean{Value: false},
												Body: []ast.Node{

													ast.Next{},
												},
											},
										},
									},
								},
							},
						}))
					})
				})
			})
		})
	})

	Describe("a normal file you might parse", func() {
		BeforeEach(func() {
			lexer = parser.NewLexer(`
#!/usr/bin/env ruby

$:.unshift File.expand_path('../../lib', __FILE__)

require 'mspec/commands/mspec-run'
require 'mspec'

MSpecRun.main
`)
		})

		It("is parsed just fine", func() {
			Expect(parser.RubyParse(lexer)).To(BeSuccessful())
			Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).To(BeNil())
			Expect(len(parser.Statements)).To(Equal(4))
		})
	})

	Describe("more complicated expressions", func() {
		Context("bit shift, ternary and trailing 'if'", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
s << (short ? ", " : "  ") if long
`)
			})

			It("is parsed correctly", func() {
				Expect(parser.RubyParse(lexer)).To(BeSuccessful())
				Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).To(BeNil())
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.IfBlock{
						Condition: ast.BareReference{Name: "long"},
						Body: []ast.Node{
							ast.CallExpression{
								Target: ast.BareReference{Name: "s"},
								Func:   ast.BareReference{Name: "<<"},
								Args: []ast.Node{
									ast.Group{
										Body: []ast.Node{
											ast.Ternary{
												Condition: ast.BareReference{Name: "short"},
												True:      ast.InterpolatedString{Value: ", "},
												False:     ast.InterpolatedString{Value: "  "},
											},
										},
									},
								},
							},
						},
					},
				}))
			})
		})

		Context("conditionals around assignment to a call expression", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
unless option = match?(opt)
end`)
			})

			It("is parsed correctly", func() {
				Expect(parser.RubyParse(lexer)).To(BeSuccessful())
				Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).To(BeNil())
				Expect(parser.Statements).To(Equal([]ast.Node{
					ast.IfBlock{
						Condition: ast.Negation{
							Target: ast.Assignment{
								LHS: ast.BareReference{Name: "option"},
								RHS: ast.CallExpression{
									Func: ast.BareReference{Name: "match?"},
									Args: []ast.Node{ast.BareReference{Name: "opt"}},
								},
							},
						},
						Body: []ast.Node{},
					},
				}))
			})
		})
	})

	Describe("having tons of optional whitespace", func() {
		BeforeEach(func() {
			lexer = parser.NewLexer(`
class Foo<Bar
	 1+1
   5    +    5
	 puts      'foo'
	 puts(     'foo',    'bar', 'baz'    )

	 def    something
	           puts 'whatever'
               puts 'fasciculated-stripe'
	  end

	    def something2(  foo  ,   bar   )
        foo ||
          bar
        foo &&
          bar


      	end

  abc    =   123

  !  true
  ~    true
  +   5
  -    123

  a = 5 or
    false
  b = a and
    true
end

with_a_block { |foo| puts foo.inspect } # comment goes here
func.with_a_block { |foo | puts foo.inspect    } # all the comments # yep
`)
		})

		It("parses just fine", func() {
			Expect(parser.RubyParse(lexer)).To(BeSuccessful())
			Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).To(BeNil())
		})
	})

	Describe("invalid ruby", func() {
		JustBeforeEach(func() {
			Expect(parser.RubyParse(lexer)).ToNot(BeSuccessful())
			Expect(lexer.(*parser.ConcreteStatefulRubyLexer).LastError).To(HaveOccurred())
		})

		Context("given a class name that starts with a lowercase character", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
class foo
end
`)
			})

			It("fails and returns a useful parse error", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})

		PContext("when the 'next' keyword is outside of a loop or block", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("next")
			})

			It("fails to parse", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})

		XContext("when the 'break' keyword is outside of a loop or block", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("break")
			})

			It("fails to parse", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})

		PContext("when the 'return' keyword is outside of a method", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("return 5")
			})

			It("fails to parse", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})

		PContext("when a method is provided two blocks", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer(`
takes_a_block(&baz) do
  puts 'FAIL'
end
`)
			})

			It("is fails to parse", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})

		PContext("when a splat precedes a reference outside of a call expression or method declaration", func() {
			BeforeEach(func() {
				lexer = parser.NewLexer("*[1,2,3]")
			})

			It("is fails to parse", func() {
				Expect(parser.Statements).To(BeEmpty())
			})
		})
	})
})
