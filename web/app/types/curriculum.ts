export type ModuleStatus = 'draft' | 'active' | 'archived';
export type ModuleDisplayStatus = ModuleStatus | 'completed';
export type VariantStatus = 'draft' | 'approved';
export type ModuleScope = 'all' | 'active' | 'draft' | 'archived' | 'mine' | 'team';
export type OpenVariantMode = 'all' | 'groups' | 'users';

export interface Module {
  id: string;
  title: string;
  description: string;
  status: ModuleStatus;
  success_threshold: number;
  created_by: string;
  created_at: string;
  lesson_count?: number;
  assigned_count?: number;
  assigned_label?: string;
  assigned_groups?: number;
  assigned_users?: number;
  opened_done?: number;
  opened_total?: number;
  success_rate?: number | null;
}

export interface Lesson {
  id: string;
  module_id: string;
  title: string;
  position: number;
  duration_seconds?: number;
  archived_at?: string;
  created_at: string;
  variant_count?: number;
  ticket_count?: number;
  variants_label?: string;
  opened_for?: number;
  opened_total?: number;
  passed_rate?: number | null;
  avg_success?: number | null;
  attention?: string;
}

export interface Variant {
  id: string;
  lesson_id: string;
  title: string;
  position: number;
  status: VariantStatus;
  is_primary: boolean;
  created_at: string;
  ticket_count?: number;
  attempt_count?: number;
  avg_success?: number | null;
  hardest_ticket_title?: string;
  hardest_ticket_rate?: number | null;
}

export interface Ticket {
  id: string;
  variant_id?: string;
  topic_id: string;
  title: string;
  body: string;
  created_by: string;
  created_at: string;
  audio_status: string;
  incident_type_code?: string;
  incident_type?: string;
  slots_total?: number;
  slots_required?: number;
  variant_usage?: number;
  card_status?: string;
}

export interface AssignedLesson {
  lesson: Lesson;
  variant: Variant;
  attempt_id?: string;
  status?: string;
  available_from?: string;
  deadline_at?: string;
  score?: number;
}

export interface AssignedModule {
  module: Module;
  lessons: AssignedLesson[];
}

export interface AssignmentSummary {
  total_users: number;
  groups: Array<{ label: string; count: number }>;
  individuals: number;
}

export interface ModuleSummary {
  module: Module;
  lessons: Lesson[];
  assignment?: AssignmentSummary;
  attention?: string[];
}

export interface AttemptAnswer {
  incident_type_code?: string | null;
  tag_codes: string[];
  service_codes: string[];
  applicant_last_name: string;
  applicant_first_name: string;
  caller_number: string;
  dictated_number: string;
  notes: string;
}

export interface ReportError {
  field: string;
  expected?: unknown;
  actual?: unknown;
}

export interface ReportItem {
  ticket_id: string;
  score: number;
  errors: ReportError[];
}

export interface AttemptReport {
  overall_score: number;
  items: ReportItem[];
}

export interface Attempt {
  id: string;
  variant_id: string;
  user_id: string;
  granted_by: string;
  attempt_no: number;
  status: string;
  available_from?: string;
  started_at?: string;
  deadline_at?: string;
  finished_at?: string;
  score?: number;
  answers?: Record<string, AttemptAnswer>;
  report?: AttemptReport;
}

export interface ReferenceAnswer {
  ticket_id: string;
  incident_type_code: string;
  tag_codes: string[];
  service_codes: string[];
  applicant_last_name: string;
  applicant_first_name: string;
  caller_number: string;
  dictated_number: string;
}

export interface SaveAnswerBody {
  incident_type_code?: string | null;
  tag_codes: string[];
  service_codes: string[];
  applicant_last_name: string;
  applicant_first_name: string;
  caller_number: string;
  dictated_number: string;
  notes: string;
}

export interface Topic {
  id: string;
  title: string;
  created_by: string;
  created_at: string;
}

export interface Article {
  id: string;
  topic_id: string;
  title: string;
  body_md: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IncidentType {
  code: string;
  title: string;
}

export interface IncidentTag {
  code: string;
  title: string;
  sort_order: number;
}

export interface TagGroup {
  code: string;
  title: string;
  selection_mode: 'single' | 'multi' | string;
  parent_tag_code?: string;
  sort_order: number;
  tags: IncidentTag[];
}

export interface EmergencyService {
  code: string;
  title: string;
}

export interface CreateModuleBody {
  title: string;
  description: string;
}

export interface UpdateModuleBody {
  title: string;
  description: string;
  status?: ModuleStatus;
  success_threshold?: number;
}

export interface CreateLessonBody {
  title: string;
  position: number;
  duration_seconds?: number;
}

export interface CreateVariantBody {
  title: string;
  position: number;
}

export interface UpdateVariantBody {
  title: string;
  position: number;
  status?: VariantStatus;
  is_primary?: boolean;
}

export interface CreateTicketBody {
  topic_id: string;
  title: string;
  body: string;
}

export interface OpenVariantBody {
  mode: OpenVariantMode;
  module_id?: string;
  group_ids?: string[];
  user_ids?: string[];
  available_from?: string;
  deadline_at?: string;
}

export interface GrantAttemptBody {
  variant_id: string;
  available_from?: string;
  deadline_at?: string;
}

export interface AssignModuleBody {
  module_id: string;
  lessons: Array<{ lesson_id: string; variant_id: string }>;
}
