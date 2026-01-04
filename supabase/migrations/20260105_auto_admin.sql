-- Create a function to handle new user signups
create or replace function public.handle_new_user()
returns trigger as $$
declare
  user_count int;
begin
  -- Check how many users exist (including this new one)
  select count(*) into user_count from auth.users;

  -- If this is the first user, grant them the admin role
  if user_count = 1 then
    update auth.users
    set raw_app_meta_data = jsonb_set(coalesce(raw_app_meta_data, '{}'::jsonb), '{roles}', '["admin"]'::jsonb)
    where id = new.id;
  end if;

  return new;
end;
$$ language plpgsql security definer;

-- Create the trigger
create or replace trigger on_auth_user_created
  after insert on auth.users
  for each row
  execute procedure public.handle_new_user();
