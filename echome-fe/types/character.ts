export interface Character {
  id: string;
  name: string;
  description: string | null;
  prompt: string;
  avatar: string;
  voice: string | null;
  audio_example: string | null;
  flag: boolean;
  status: number;
  created_at: string;
  updated_at: string;
}
