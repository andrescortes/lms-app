CREATE TABLE IF NOT EXISTS cefr_levels (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    rank_order INTEGER NOT NULL UNIQUE
);

INSERT OR IGNORE INTO cefr_levels (code, name, description, rank_order) VALUES
    ('A1', 'Beginner', 'Understand and use familiar everyday expressions and very basic phrases.', 1),
    ('A2', 'Elementary', 'Communicate in simple and routine tasks requiring a direct exchange of information.', 2),
    ('B1', 'Intermediate', 'Deal with most situations likely to arise while traveling or discussing familiar topics.', 3),
    ('B2', 'Upper Intermediate', 'Interact with a degree of fluency and spontaneity with native speakers.', 4),
    ('C1', 'Advanced', 'Express ideas fluently and spontaneously without much obvious searching for expressions.', 5),
    ('C2', 'Proficiency', 'Understand with ease virtually everything heard or read and summarize complex arguments.', 6);