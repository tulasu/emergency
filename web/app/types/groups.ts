export interface GroupMember {
  user_id: string;
  role: 'teacher' | 'student' | string;
}

export interface Group {
  id: string;
  name: string;
  members?: GroupMember[];
}

export interface UserProfile {
  id: string;
  login: string;
  role: string;
  full_name: string;
  blocked?: boolean;
  blocked_at?: string;
  created_at?: string;
}
