package appinit_test

import (
	"reflect"
	"testing"

	"github.com/momo182/ssup/src/entity"
	appinit "github.com/momo182/ssup/src/usecase/appinit"
)

func TestMergeVars(t *testing.T) {
	type args struct {
		conf    *entity.Supfile
		network *entity.Network
	}
	tests := []struct {
		name string
		args args
		want entity.EnvList
	}{
		{
			name: "Both conf and network have unique vars",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR3", "value3")
						e.Set("VAR4", "value4")
						return e
					}(),
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				e.Set("VAR3", "value3")
				e.Set("VAR4", "value4")
				return e
			}(),
		},
		{
			name: "Conf vars override network vars",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "overridden")
						e.Set("VAR3", "value3")
						return e
					}(),
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "overridden") // VAR1 from conf should override network's VAR1
				e.Set("VAR2", "value2")
				e.Set("VAR3", "value3")
				return e
			}(),
		},
		{
			name: "Empty conf vars",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						return entity.EnvList{}
					}(),
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Empty network vars",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						return entity.EnvList{}
					}(),
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Both conf and network vars are empty",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						return entity.EnvList{}
					}(),
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						return entity.EnvList{}
					}(),
				},
			},
			want: func() entity.EnvList {
				return entity.EnvList{}
			}(),
		},
		{
			name: "Nil conf vars",
			args: args{
				conf: &entity.Supfile{
					Env: entity.EnvList{},
				},
				network: &entity.Network{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Nil network vars",
			args: args{
				conf: &entity.Supfile{
					Env: func() entity.EnvList {
						e := entity.EnvList{}
						e.Set("VAR1", "value1")
						e.Set("VAR2", "value2")
						return e
					}(),
				},
				network: &entity.Network{
					Env: entity.EnvList{},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Both conf and network vars are nil",
			args: args{
				conf: &entity.Supfile{
					Env: entity.EnvList{},
				},
				network: &entity.Network{
					Env: entity.EnvList{},
				},
			},
			want: func() entity.EnvList {
				return entity.EnvList{}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appinit.MergeVars(tt.args.conf, tt.args.network); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetEnvValues(t *testing.T) {
	type args struct {
		vars        entity.EnvList
		initialArgs *entity.InitialArgs
	}
	tests := []struct {
		name string
		args args
		want entity.EnvList
	}{
		{
			name: "No initial args",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					e.Set("VAR2", "value2")
					return e
				}(),
				initialArgs: &entity.InitialArgs{},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "With initial args",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					e.Set("VAR2", "value2")
					return e
				}(),
				initialArgs: &entity.InitialArgs{
					EnvVars: entity.FlagStringSlice{"VAR3=value3", "VAR4=value4"},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				e.Set("VAR3", "value3")
				e.Set("VAR4", "value4")
				return e
			}(),
		},
		{
			name: "Overlapping env vars",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					e.Set("VAR2", "value2")
					return e
				}(),
				initialArgs: &entity.InitialArgs{
					EnvVars: entity.FlagStringSlice{"VAR2=value3", "VAR4=value4"},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value3")
				e.Set("VAR4", "value4")
				return e
			}(),
		},
		{
			name: "Empty env vars",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					return e
				}(),
				initialArgs: &entity.InitialArgs{
					EnvVars: entity.FlagStringSlice{"VAR1=value1", "VAR2=value2"},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Empty initial args",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					e.Set("VAR2", "value2")
					return e
				}(),
				initialArgs: &entity.InitialArgs{},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Both empty",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					return e
				}(),
				initialArgs: &entity.InitialArgs{},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				return e
			}(),
		},
		{
			name: "Initial args with malformed env vars",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					return e
				}(),
				initialArgs: &entity.InitialArgs{
					EnvVars: entity.FlagStringSlice{"MALFORMED", "VAR2=value2"},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
		{
			name: "Initial args with empty env vars",
			args: args{
				vars: func() entity.EnvList {
					e := entity.EnvList{}
					e.Set("VAR1", "value1")
					return e
				}(),
				initialArgs: &entity.InitialArgs{
					EnvVars: entity.FlagStringSlice{"", "VAR2=value2"},
				},
			},
			want: func() entity.EnvList {
				e := entity.EnvList{}
				e.Set("VAR1", "value1")
				e.Set("VAR2", "value2")
				return e
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appinit.SetEnvValues(&tt.args.vars, tt.args.initialArgs)
			if !reflect.DeepEqual(tt.args.vars, tt.want) {
				t.Errorf("SetEnvValues() = %v, want %v", tt.args.vars, tt.want)
			}
		})
	}
}
