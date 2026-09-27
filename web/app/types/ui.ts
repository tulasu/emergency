export type TbButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'icon';
export type TbButtonWidth = 'auto' | 'block';

export interface TbSelectOption {
  value: string;
  label: string;
}

export type TbIconName =
  | 'arrow-left'
  | 'plus'
  | 'layers'
  | 'folder'
  | 'users'
  | 'user-plus'
  | 'settings'
  | 'log-out'
  | 'user'
  | 'copy'
  | 'chevron-down'
  | 'chevron-right'
  | 'check'
  | 'x'
  | 'file'
  | 'search'
  | 'chart'
  | 'book'
  | 'home'
  | 'archive'
  | 'calendar'
  | 'pencil'
  | 'play'
  | 'clock'
  | 'lock'
  | 'info';

export type TbBadgeTone = 'success' | 'warning' | 'info' | 'neutral' | 'danger';

export interface TbTabItem {
  id: string;
  label: string;
  count?: number;
}
