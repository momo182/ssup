package usecase_test

import (
	"testing"

	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/usecase"
	"github.com/stretchr/testify/assert"
)

func TestAppendCommandEnvsToTask(t *testing.T) {
	t.Run("Test Null Command", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic but did not get one")
			}
		}()
		usecase.AppendCommandEnvsToTask(nil, &entity.Task{})
	})

	t.Run("Test Null Task", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic but did not get one")
			}
		}()
		usecase.AppendCommandEnvsToTask(&entity.Command{}, nil)
	})

	t.Run("Test Command with No Envs", func(t *testing.T) {
		task := &entity.Task{}
		cmd := &entity.Command{}
		usecase.AppendCommandEnvsToTask(cmd, task)
		assert.Equal(t, 0, len(task.Env.Keys()))
	})

	t.Run("Test Command with Non-Empty Envs", func(t *testing.T) {
		task := &entity.Task{}
		cmd := &entity.Command{}
		cmd.Env.Set("VAR1", "value1")
		cmd.Env.Set("VAR2", "value2")
		usecase.AppendCommandEnvsToTask(cmd, task)
		assert.Equal(t, "value1", task.Env.Get("VAR1"))
		assert.Equal(t, "value2", task.Env.Get("VAR2"))
	})

	t.Run("Test Overwriting Existing Task Env Variables", func(t *testing.T) {
		task := &entity.Task{}
		task.Env.Set("VAR1", "old_value1")
		task.Env.Set("VAR2", "value2")
		cmd := &entity.Command{}
		cmd.Env.Set("VAR1", "new_value1")
		usecase.AppendCommandEnvsToTask(cmd, task)
		assert.Equal(t, "new_value1", task.Env.Get("VAR1"))
		assert.Equal(t, "value2", task.Env.Get("VAR2"))
	})

	t.Run("Test Empty Command and Task Envs", func(t *testing.T) {
		task := &entity.Task{}
		cmd := &entity.Command{}
		usecase.AppendCommandEnvsToTask(cmd, task)
		assert.Equal(t, 0, len(task.Env.Keys()))
	})

	t.Run("Test Complex Envs", func(t *testing.T) {
		task := &entity.Task{}
		task.Env.Set("EXISTING_KEY", "existing_value")
		cmd := &entity.Command{}
		cmd.Env.Set("COMPLEX_KEY_1", "value with spaces")
		cmd.Env.Set("COMPLEX_KEY_2", "value_with_special_chars!@#$%^&*()")
		usecase.AppendCommandEnvsToTask(cmd, task)
		assert.Equal(t, "existing_value", task.Env.Get("EXISTING_KEY"))
		assert.Equal(t, "value with spaces", task.Env.Get("COMPLEX_KEY_1"))
		assert.Equal(t, "value_with_special_chars!@#$%^&*()", task.Env.Get("COMPLEX_KEY_2"))
	})
}
