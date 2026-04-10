INSERT INTO users (id, name, email, password)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Test User',
    'test@example.com',
    crypt('password123', gen_salt('bf', 12))
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO projects (id, name, description, owner_id)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Greening India Rollout',
    'Initial seeded project for evaluator testing',
    '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, creator_id, due_date)
VALUES
    (
        '33333333-3333-3333-3333-333333333331',
        'Draft program brief',
        'Define scope and stakeholders',
        'todo',
        'high',
        '22222222-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_DATE + INTERVAL '7 days'
    ),
    (
        '33333333-3333-3333-3333-333333333332',
        'Collect district data',
        'Gather baseline environmental metrics',
        'in_progress',
        'medium',
        '22222222-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_DATE + INTERVAL '14 days'
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        'Publish pilot report',
        'Share outcomes with policy team',
        'done',
        'low',
        '22222222-2222-2222-2222-222222222222',
        NULL,
        '11111111-1111-1111-1111-111111111111',
        NULL
    )
ON CONFLICT (id) DO NOTHING;
