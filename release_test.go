// Copyright 2021 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReleaseConfig(t *testing.T) {
	testcases := []struct {
		name        string
		configFile  string
		expected    *ReleaseConfig
		expectedErr error
	}{
		{
			name:        "empty config",
			configFile:  "testdata/empty-config.txt",
			expectedErr: fmt.Errorf("tag must be specified"),
		},
		{
			name:       "valid config",
			configFile: "testdata/valid-config.txt",
			expected: &ReleaseConfig{
				Tag: "v1.1.0",
				Name: "hello",
				CommitInclude: ReleaseCommitMatcherConfig{
					Contains: []string{
						"app/hello",
					},
				},
				CommitExclude: ReleaseCommitMatcherConfig{
					Prefixes: []string{
						"Merge pull request #",
					},
				},
				CommitCategories: []ReleaseCommitCategoryConfig{
					ReleaseCommitCategoryConfig{
						Id:    "_category_0",
						Title: "Breaking Changes",
						ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
							Contains: []string{"change-category/breaking-change"},
						},
					},
					ReleaseCommitCategoryConfig{
						Id:    "_category_1",
						Title: "New Features",
						ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
							Contains: []string{"change-category/new-feature"},
						},
					},
					ReleaseCommitCategoryConfig{
						Id:    "_category_2",
						Title: "Notable Changes",
						ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
							Contains: []string{"change-category/notable-change"},
						},
					},
					ReleaseCommitCategoryConfig{
						Id:                         "_category_3",
						Title:                      "Internal Changes",
						ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{},
					},
				},
				ReleaseNoteGenerator: ReleaseNoteGeneratorConfig{
					ShowCommitter:       true,
					UseReleaseNoteBlock: true,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := testdata.ReadFile(tc.configFile)
			require.NoError(t, err)

			cfg, err := parseReleaseConfig(data)
			assert.Equal(t, tc.expected, cfg)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestBuildReleaseCommits(t *testing.T) {
	config := ReleaseConfig{
		Tag: "v1.1.0",
		Name: "hello",
		CommitInclude: ReleaseCommitMatcherConfig{
			Contains: []string{
				"app/hello",
			},
		},
		CommitExclude: ReleaseCommitMatcherConfig{
			Prefixes: []string{
				"Merge pull request #",
			},
		},
		CommitCategories: []ReleaseCommitCategoryConfig{
			ReleaseCommitCategoryConfig{
				Id:    "breaking-change",
				Title: "Breaking Changes",
				ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
					Contains: []string{"change-category/breaking-change"},
				},
			},
			ReleaseCommitCategoryConfig{
				Id:    "new-feature",
				Title: "New Features",
				ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
					Contains: []string{"change-category/new-feature"},
				},
			},
			ReleaseCommitCategoryConfig{
				Id:    "notable-change",
				Title: "Notable Changes",
				ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{
					Contains: []string{"change-category/notable-change"},
				},
			},
			ReleaseCommitCategoryConfig{
				Id:                         "internal-change",
				Title:                      "Internal Changes",
				ReleaseCommitMatcherConfig: ReleaseCommitMatcherConfig{},
			},
		},
		ReleaseNoteGenerator: ReleaseNoteGeneratorConfig{
			ShowCommitter:       true,
			UseReleaseNoteBlock: true,
		},
	}

	testcases := []struct {
		name     string
		commits  []Commit
		config   ReleaseConfig
		expected []ReleaseCommit
	}{
		{
			name:     "empty",
			expected: []ReleaseCommit{},
		},
		{
			name: "ok",
			commits: []Commit{
				Commit{
					Subject: "Commit 1 message",
					Body:    "commit 1\napp/hello\n- change-category/breaking-change",
				},
				Commit{
					Subject: "Commit 2 message",
					Body:    "commit 2\napp/hello",
				},
				Commit{
					Subject: "Commit 3 message",
					Body:    "commit 3\napp/hello\n- change-category/notable-change",
				},
				Commit{
					Subject: "Commit 4 message",
					Body:    "commit 4\napp/hello\n```release-note\nCommit 4 release note\n```\n- change-category/notable-change\n",
				},
				Commit{
					Subject: "Commit 5 message",
					Body:    "commit 5",
				},
			},
			config: config,
			expected: []ReleaseCommit{
				ReleaseCommit{
					Commit: Commit{
						Subject: "Commit 1 message",
						Body:    "commit 1\napp/hello\n- change-category/breaking-change",
					},
					CategoryName: "breaking-change",
					ReleaseNote:  "Commit 1 message",
				},
				ReleaseCommit{
					Commit: Commit{
						Subject: "Commit 2 message",
						Body:    "commit 2\napp/hello",
					},
					CategoryName: "internal-change",
					ReleaseNote:  "Commit 2 message",
				},
				ReleaseCommit{
					Commit: Commit{
						Subject: "Commit 3 message",
						Body:    "commit 3\napp/hello\n- change-category/notable-change",
					},
					CategoryName: "notable-change",
					ReleaseNote:  "Commit 3 message",
				},
				ReleaseCommit{
					Commit: Commit{
						Subject: "Commit 4 message",
						Body:    "commit 4\napp/hello\n```release-note\nCommit 4 release note\n```\n- change-category/notable-change\n",
					},
					CategoryName: "notable-change",
					ReleaseNote:  "Commit 4 release note",
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildReleaseCommits(tc.commits, tc.config)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestRenderReleaseNote(t *testing.T) {
	testcases := []struct {
		name     string
		proposal ReleaseProposal
		config   ReleaseConfig
		expected string
	}{
		{
			name: "no category",
			proposal: ReleaseProposal{
				Tag:    "v0.2.0",
				PreTag: "v0.1.0",
				Commits: []ReleaseCommit{
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 1 message",
							Body:    "commit 1\n- change-category/breaking-change",
						},
						ReleaseNote: "Commit 1 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 2 message",
							Body:    "commit 2",
						},
						ReleaseNote: "Commit 2 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 3 message",
							Body:    "commit 3\n- change-category/notable-change",
						},
						ReleaseNote: "Commit 3 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 4 message",
							Body:    "commit 4\n```release-note\nCommit 4 release note\n```\n- change-category/notable-change",
						},
						ReleaseNote: "Commit 4 release note",
					},
				},
			},
			config:   ReleaseConfig{},
			expected: "testdata/no-category-release-note.txt",
		},
		{
			name: "has category",
			proposal: ReleaseProposal{
				Tag:    "v0.2.0",
				PreTag: "v0.1.0",
				Commits: []ReleaseCommit{
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 1 message",
							Body:    "commit 1\n- change-category/breaking-change",
						},
						CategoryName: "breaking-change",
						ReleaseNote:  "Commit 1 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 2 message",
							Body:    "commit 2",
						},
						CategoryName: "internal-change",
						ReleaseNote:  "Commit 2 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 3 message",
							Body:    "commit 3\n- change-category/notable-change",
						},
						CategoryName: "notable-change",
						ReleaseNote:  "Commit 3 message",
					},
					ReleaseCommit{
						Commit: Commit{
							Subject: "Commit 4 message",
							Body:    "commit 4\n```release-note\nCommit 4 release note\n```\n- change-category/notable-change",
						},
						CategoryName: "notable-change",
						ReleaseNote:  "Commit 4 release note",
					},
				},
			},
			config: ReleaseConfig{
				CommitCategories: []ReleaseCommitCategoryConfig{
					ReleaseCommitCategoryConfig{
						Id:    "breaking-change",
						Title: "Breaking Changes",
					},
					ReleaseCommitCategoryConfig{
						Id:    "new-feature",
						Title: "New Features",
					},
					ReleaseCommitCategoryConfig{
						Id:    "notable-change",
						Title: "Notable Changes",
					},
					ReleaseCommitCategoryConfig{
						Id:    "internal-change",
						Title: "Internal Changes",
					},
				},
			},
			expected: "testdata/has-category-release-note.txt",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderReleaseNote(tc.proposal, tc.config)

			expected, err := testdata.ReadFile(tc.expected)
			require.NoError(t, err)

			assert.Equal(t, string(expected), string(got))
		})
	}
}
