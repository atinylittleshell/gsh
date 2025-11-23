package evaluate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		name     string
		cmd1     string
		cmd2     string
		minScore float64
		maxScore float64
		wantErr  bool
		debug    bool
	}{
		{
			name:     "identical simple commands",
			cmd1:     "ls -l",
			cmd2:     "ls -l",
			minScore: 1.0,
			maxScore: 1.0,
			wantErr:  false,
		},
		{
			name:     "similar simple commands with different flags",
			cmd1:     "ls -l",
			cmd2:     "ls -la",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "completely different simple commands",
			cmd1:     "ls -l",
			cmd2:     "pwd",
			minScore: 0.0,
			maxScore: 0.3,
			wantErr:  false,
		},
		{
			name:     "similar commands with different arguments",
			cmd1:     "git commit -m 'test'",
			cmd2:     "git commit -m 'different message'",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "pipe commands identical",
			cmd1:     "ls -l | grep .txt | sort",
			cmd2:     "ls -l | grep .txt | sort",
			minScore: 1.0,
			maxScore: 1.0,
			wantErr:  false,
		},
		{
			name:     "pipe commands similar",
			cmd1:     "ls -l | grep .txt | sort",
			cmd2:     "ls -la | grep .txt | sort -r",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "pipe vs no pipe",
			cmd1:     "ls -l | grep .txt",
			cmd2:     "ls -l",
			minScore: 0.02,
			maxScore: 0.5,
			wantErr:  false,
		},
		{
			name:     "commands with environment variables",
			cmd1:     "echo $HOME",
			cmd2:     "echo $PWD",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with subshells",
			cmd1:     "echo $(date)",
			cmd2:     "echo $(pwd)",
			minScore: 0.4,
			maxScore: 0.7,
			wantErr:  false,
		},
		{
			name:     "different commands with different subshells",
			cmd1:     "echo $(date)",
			cmd2:     "ls $(pwd)",
			minScore: 0,
			maxScore: 0.3,
			wantErr:  false,
		},
		{
			name:     "commands with redirections identical",
			cmd1:     "echo 'test' > file.txt",
			cmd2:     "echo 'test' > file.txt",
			minScore: 1.0,
			maxScore: 1.0,
			wantErr:  false,
		},
		{
			name:     "commands with different redirections",
			cmd1:     "echo 'test' > file.txt",
			cmd2:     "echo 'test' >> file.txt",
			minScore: 0.7,
			maxScore: 0.95,
			wantErr:  false,
		},
		{
			name:     "commands with quotes vs no quotes",
			cmd1:     "echo 'hello world'",
			cmd2:     "echo hello world",
			minScore: 0.6,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "complex commands with multiple features",
			cmd1:     "VAR=123 echo $(date) | grep 2023 > output.txt",
			cmd2:     "VAR=456 echo $(date) | grep 2024 > output.txt",
			minScore: 0.7,
			maxScore: 0.95,
			wantErr:  false,
		},
		{
			name:     "empty strings",
			cmd1:     "",
			cmd2:     "",
			minScore: 0.0,
			maxScore: 0.0,
			wantErr:  true,
		},
		{
			name:     "empty vs non-empty",
			cmd1:     "",
			cmd2:     "ls",
			minScore: 0.0,
			maxScore: 0.0,
			wantErr:  true,
		},
		{
			name:     "commands with glob patterns",
			cmd1:     "ls *.txt",
			cmd2:     "ls *.md",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with different glob patterns",
			cmd1:     "ls *.txt",
			cmd2:     "ls test/*/*.md",
			minScore: 0.5,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with background execution",
			cmd1:     "sleep 10 &",
			cmd2:     "sleep 20 &",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "invalid shell syntax",
			cmd1:     "ls (",
			cmd2:     "ls",
			minScore: 0.0,
			maxScore: 0.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.debug {
				debug = true
				defer func() { debug = false }()
			}

			score, err := SimilarityScore(tt.cmd1, tt.cmd2)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, score, tt.minScore, "similarity score should be >= %v", tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore, "similarity score should be <= %v", tt.maxScore)
		})
	}
}

func TestSimilarityScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cmd1     string
		cmd2     string
		minScore float64
		maxScore float64
		wantErr  bool
		debug    bool
	}{
		{
			name:     "very long commands",
			cmd1:     "find . -type f -name '*.go' -not -path './vendor/*' -exec grep -l 'TODO' {} \\; | xargs wc -l | sort -n",
			cmd2:     "find . -type f -name '*.rs' -not -path './target/*' -exec grep -l 'TODO' {} \\; | xargs wc -l | sort -n",
			minScore: 0.9,
			maxScore: 0.99,
			wantErr:  false,
		},
		{
			name:     "commands with unicode",
			cmd1:     "echo '你好'",
			cmd2:     "echo 'hello'",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with multiple spaces",
			cmd1:     "ls   -l     -a",
			cmd2:     "ls -l -a",
			minScore: 0.9,
			maxScore: 1.0,
			wantErr:  false,
		},
		{
			name:     "commands with escaped characters",
			cmd1:     "echo \"hello\\nworld\"",
			cmd2:     "echo \"hello\\tworld\"",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with heredoc",
			cmd1:     "cat << EOF\nhello\nEOF",
			cmd2:     "cat << EOF\nworld\nEOF",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
		{
			name:     "commands with process substitution",
			cmd1:     "diff <(ls dir1) <(ls dir2)",
			cmd2:     "diff <(ls dir1) <(ls dir3)",
			minScore: 0.8,
			maxScore: 1.0,
			wantErr:  false,
		},
		{
			name:     "commands with arithmetic expansion",
			cmd1:     "echo $((1 + 2))",
			cmd2:     "echo $((3 + 4))",
			minScore: 0.7,
			maxScore: 0.9,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.debug {
				debug = true
				defer func() { debug = false }()
			}

			score, err := SimilarityScore(tt.cmd1, tt.cmd2)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, score, tt.minScore, "similarity score should be >= %v", tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore, "similarity score should be <= %v", tt.maxScore)
		})
	}
}
