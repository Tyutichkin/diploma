import { OptimizedRoute } from "../../types/task";
import { Card, CardContent } from "../ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { Clock, Info, Navigation, Route, AudioWaveform } from "lucide-react";

interface OptimizedRouteSummaryProps {
    info: OptimizedRoute;
}

interface MetricProps {
    icon: React.ReactNode;
    label: string;
    value: string;
    hint: string;
}

function Metric({ icon, label, value, hint }: MetricProps) {
    return (
        <div className="flex items-center gap-2">
            {icon}
            <span className="text-gray-700">
                {label}: <strong>{value}</strong>
            </span>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                </TooltipTrigger>
                <TooltipContent>
                    <p>{hint}</p>
                </TooltipContent>
            </Tooltip>
        </div>
    );
}

export function OptimizedRouteSummary({ info }: OptimizedRouteSummaryProps) {
    return (
        <Card className="mb-6 border-blue-200 bg-blue-50">
            <CardContent className="p-4">
                <div className="flex items-start gap-3">
                    <Info className="mt-0.5 h-5 w-5 flex-shrink-0 text-blue-600" />
                    <div className="flex-1">
                        <h3 className="mb-2 font-semibold text-blue-900">
                            Информация о маршруте
                        </h3>
                        <div className="grid grid-cols-1 gap-4 text-sm md:grid-cols-3">
                            <Metric
                                icon={
                                    <Navigation className="h-4 w-4 text-blue-600" />
                                }
                                label="Расстояние"
                                value={`${info.totalDistance} км`}
                                hint="Суммарное расстояние по дорогам между всеми точками маршрута"
                            />
                            <Metric
                                icon={
                                    <Route className="h-4 w-4 text-blue-600" />
                                }
                                label="Время в пути"
                                value={`${info.totalTravelTime} мин`}
                                hint="Суммарное время переездов между точками (без учёта выполнения задач)"
                            />
                            <Metric
                                icon={
                                    <AudioWaveform className="h-4 w-4 text-blue-600" />
                                }
                                label="Общее время"
                                value={`${info.totalDuration} мин`}
                                hint="Время в пути + суммарная длительность выполнения всех задач"
                            />
                        </div>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
