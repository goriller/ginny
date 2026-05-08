"""Centralized configuration for tunable parameters."""

# Knowledge lifecycle
DECAY_DAYS_THRESHOLD = 90        # Days without use before decay starts
DECAY_RATE = 0.1                 # Effectiveness reduction per decay cycle
GC_EFFECTIVENESS_THRESHOLD = 0.1  # Entries below this get garbage collected

# Knowledge retrieval
FUZZY_MATCH_THRESHOLD = 0.72     # Minimum SequenceMatcher ratio for fuzzy match (raised from 0.6 to cut cross-project noise)
RELEVANCE_WEIGHTS = {
    "trigger_match": 0.4,
    "effectiveness": 0.3,
    "recency": 0.2,
    "usage": 0.1,
}
RECENCY_DECAY_DAYS = 365.0       # Days over which recency decays to 0
USAGE_NORMALIZATION = 100.0      # Normalize usage_count by this value
TOP_K_RESULTS = 10               # Default number of results returned

# Relevance thresholds for classifying query results (used by trigger.py)
# HIGH_RELEVANCE_THRESHOLD: entries at or above this → "## 相关知识" (was hardcoded 0.45)
# MIN_RELEVANCE_THRESHOLD:  entries below this are silently excluded from all output,
#   preventing high-effectiveness cross-project entries from appearing via recency/usage alone.
HIGH_RELEVANCE_THRESHOLD = 0.65
MIN_RELEVANCE_THRESHOLD = 0.25

# Fuzzy match effectiveness weight scaling
# When a keyword entry only matched via fuzzy (match_score < 2), reduce the
# effectiveness and recency contribution so cross-project pollution is dampened.
FUZZY_MATCH_EFF_SCALE = 0.35     # effectiveness weight multiplier for fuzzy-only matches
FUZZY_MATCH_REC_SCALE = 0.50     # recency weight multiplier for fuzzy-only matches

# Summarizer
MIN_INPUT_LENGTH = 10            # Minimum text length for single-sentence validation

# Knowledge category to directory mapping (single source of truth)
CATEGORY_DIRS = {
    'experience': 'experiences',
    'tech-stack': 'tech-stacks',
    'scenario': 'scenarios',
    'problem': 'problems',
    'testing': 'testing',
    'pattern': 'patterns',
    'skill': 'skills',
}
VALID_CATEGORIES = list(CATEGORY_DIRS.keys())
