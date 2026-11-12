-- Mirrors the hired-rocks shapes the nested generator serves: accounts with users and avatars,
-- companies with avatars, vacancies with skills, connects with chats/offers/feedbacks.
CREATE TABLE images (
  id uuid PRIMARY KEY,
  url text NOT NULL
);

CREATE TABLE users (
  id uuid PRIMARY KEY,
  name text NOT NULL,
  avatar_id uuid REFERENCES images (id)
);

CREATE TABLE accounts (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users (id)
);

CREATE TABLE recruiter_accounts (
  id uuid PRIMARY KEY REFERENCES accounts (id),
  title text NOT NULL
);

CREATE TABLE jobseeker_accounts (
  id uuid PRIMARY KEY REFERENCES accounts (id),
  headline text NOT NULL
);

CREATE TABLE companies (
  id uuid PRIMARY KEY,
  name text NOT NULL,
  avatar_id uuid REFERENCES images (id)
);

CREATE TABLE skills (
  id uuid PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE vacancies (
  id uuid PRIMARY KEY,
  title text NOT NULL,
  author_id uuid NOT NULL REFERENCES recruiter_accounts (id),
  company_id uuid NOT NULL REFERENCES companies (id)
);

CREATE TABLE vacancy_skills (
  id uuid PRIMARY KEY,
  vacancy_id uuid NOT NULL REFERENCES vacancies (id),
  skill_id uuid NOT NULL REFERENCES skills (id)
);

CREATE TABLE connects (
  id uuid PRIMARY KEY,
  vacancy_id uuid NOT NULL REFERENCES vacancies (id),
  jobseeker_account_id uuid NOT NULL REFERENCES jobseeker_accounts (id),
  recruiter_account_id uuid NOT NULL REFERENCES recruiter_accounts (id),
  interviewer_account_id uuid REFERENCES recruiter_accounts (id)
);

CREATE TABLE chats (
  id uuid PRIMARY KEY,
  connect_id uuid NOT NULL REFERENCES connects (id),
  is_private boolean NOT NULL DEFAULT false
);

CREATE TABLE chat_members (
  id uuid PRIMARY KEY,
  chat_id uuid NOT NULL REFERENCES chats (id),
  account_id uuid NOT NULL REFERENCES accounts (id)
);

CREATE TABLE offers (
  id uuid PRIMARY KEY,
  connect_id uuid NOT NULL REFERENCES connects (id),
  amount integer NOT NULL
);

CREATE TABLE feedbacks (
  id uuid PRIMARY KEY,
  connect_id uuid NOT NULL REFERENCES connects (id),
  body text NOT NULL
);
