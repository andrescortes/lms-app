import { CEFRCode } from './auth.model';

export interface CEFRLevel {
  code: CEFRCode;
  name: string;
  description: string;
  rank_order: number;
  unlocked: boolean;
  is_current: boolean;
}

export interface LevelsResponse {
  user_id: string;
  current_level: CEFRCode;
  levels: CEFRLevel[];
}
