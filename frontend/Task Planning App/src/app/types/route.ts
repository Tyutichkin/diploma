export interface SavedRouteSummary {
  id: string;
  algorithm: string;
  computedAt: string;
}

export interface RouteStopTiming {
  taskId: string;
  position: number;
  travelFromPrevSec?: number;
  arriveTime?: string;       // ISO 8601
  serviceStartTime?: string; // ISO 8601
  serviceEndTime?: string;   // ISO 8601
  waitSec?: number;
}

export interface SavedRouteDetails extends SavedRouteSummary {
  orderedTaskIds: string[];
  stops?: RouteStopTiming[];
  totalDistanceKm?: number;
  totalTravelTimeMin?: number;
  totalDurationMin?: number;
}
