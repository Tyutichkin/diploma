export interface Task {
  id: string;
  title: string;
  address: string;
  latitude?: number;
  longitude?: number;
  duration: number; // minutes
  timeWindowStart?: string; // ISO time string or "HH:mm"
  timeWindowEnd?: string; // ISO time string or "HH:mm"
  priority?: number; // 1-5
  completed?: boolean;
  order?: number;
}

export interface OptimizedRoute {
  tasks: Task[];
  totalDistance: number; // km
  totalDuration: number; // minutes
  totalTravelTime: number; // minutes
}
