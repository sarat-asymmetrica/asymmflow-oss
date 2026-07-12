
export interface Quaternion {
  W: number;
  X: number;
  Y: number;
  Z: number;
}

export interface Keyframe {
  time: number;
  value: Quaternion | number | string | any;
  easing?: string;
}

export interface Track {
  name: string;
  type: string;
  interpolation: string;
  keyframes: Keyframe[];
}

export interface Geometry {
  type: string;
  scale: number;
  color: string;
}

export interface Metadata {
  title: string;
  author: string;
  description?: string;
  duration: number;
  fps: number;
}

export interface QGIF {
  version: string;
  metadata: Metadata;
  tracks: Track[];
  geometry: Geometry;
}
