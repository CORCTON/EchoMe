import { create } from "zustand";

export interface FileObject {
  url: string;
  name: string;
  type: string;
}

interface FileState {
  files: FileObject[];
  addFile: (file: FileObject) => void;
  setFiles: (files: FileObject[]) => void;
  clearFiles: () => void;
}

export const useFileStore = create<FileState>((set) => ({
  files: [],
  addFile: (file) => set((state) => ({ files: [...state.files, file] })),
  setFiles: (files) => set({ files }),
  clearFiles: () => set({ files: [] }),
}));
