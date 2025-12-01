-- Users
INSERT INTO users (email, password_hash, email_verified) VALUES 
('alice@example.com', '$2a$10$X7.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1', true),
('bob@example.com', '$2a$10$X7.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1', true),
('charlie@example.com', '$2a$10$X7.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1', true);

-- Blogs
INSERT INTO blogs (user_id, title, description, subdomain, nav, is_list_reader) VALUES 
(1, 'Alice Wonderland', 'Thoughts from the rabbit hole', 'alice', '[Home](/) [About](/p/about)', true),
(2, 'Bob Builder', 'Can we fix it?', 'bob', '[Projects](/p/projects)', true),
(3, 'Charlie Chef', 'Cooking with Charlie', 'charlie', '[Recipes](/t/recipes)', false);

-- Tags
INSERT INTO tags (title) VALUES 
('Technology'), ('Cooking'), ('Life'), ('Travel'), ('Coding');

-- Posts (Alice)
INSERT INTO posts (blog_id, title, slug, content, draft, created_at, updated_at) VALUES 
(1, 'Down the Rabbit Hole', 'down-the-rabbit-hole', '# Down the Rabbit Hole\n\nIt was a sunny day...', false, datetime('now', '-5 days'), datetime('now', '-5 days')),
(1, 'Tea Party', 'tea-party', '# Tea Party\n\nThe Hatter was mad as usual.', false, datetime('now', '-2 days'), datetime('now', '-2 days')),
(1, 'Draft Post', 'draft-post', '# Draft\n\nThis should not be visible.', true, datetime('now'), datetime('now'));

-- Posts (Bob)
INSERT INTO posts (blog_id, title, slug, content, draft, created_at, updated_at) VALUES 
(2, 'Building a Deck', 'building-a-deck', '# Building a Deck\n\nFirst, measure twice.', false, datetime('now', '-3 days'), datetime('now', '-3 days')),
(2, 'Fixing the Roof', 'fixing-the-roof', '# Fixing the Roof\n\nWatch out for rain.', false, datetime('now', '-1 days'), datetime('now', '-1 days'));

-- Posts (Charlie)
INSERT INTO posts (blog_id, title, slug, content, draft, created_at, updated_at) VALUES 
(3, 'Spicy Curry', 'spicy-curry', '# Spicy Curry\n\nIngredients:\n- Curry paste\n- Coconut milk', false, datetime('now', '-4 days'), datetime('now', '-4 days'));

-- Post Tags
-- Alice's posts
INSERT INTO post_tags (post_id, tag_id) VALUES 
(1, 3), -- Down the Rabbit Hole -> Life
(2, 3); -- Tea Party -> Life

-- Bob's posts
INSERT INTO post_tags (post_id, tag_id) VALUES 
(4, 1), -- Building a Deck -> Technology (sort of)
(4, 5), -- Building a Deck -> Coding (maybe he coded the plans?)
(5, 3); -- Fixing the Roof -> Life

-- Charlie's posts
INSERT INTO post_tags (post_id, tag_id) VALUES 
(6, 2); -- Spicy Curry -> Cooking

-- Pages
INSERT INTO pages (blog_id, title, slug, content, draft, created_at, updated_at) VALUES
(1, 'About Me', 'about', '# About Alice\n\nI like cats.', false, datetime('now'), datetime('now')),
(2, 'Projects', 'projects', '# My Projects\n\nList of buildings.', false, datetime('now'), datetime('now'));
