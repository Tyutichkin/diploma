import { OptimizedRoute } from "../../types/task";
import { Card, CardContent } from "../ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { Info, Navigation, Route, AudioWaveform } from "lucide-react";

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
        <div className="flex items-center gap-1.5 min-w-0 whitespace-nowrap">
            <span className="flex-shrink-0">{icon}</span>
            <span className="text-gray-700 text-xs sm:text-sm">
                <span className="text-gray-600">{label}:</span>{' '}
                <strong className="text-gray-900 font-semibold">{value}</strong>
            </span>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                </TooltipTrigger>
                <TooltipContent>
                    <p>
                        <span className="font-medium">{label}.</span> {hint}
                    </p>
                </TooltipContent>
            </Tooltip>
        </div>
    );
}

export function OptimizedRouteSummary({ info }: OptimizedRouteSummaryProps) {
    return (
        <Card className="mb-3 border-blue-200 bg-blue-50 py-0">
            <CardContent className="px-3 py-2 sm:px-4 sm:py-2.5">
                <div className="flex items-start gap-3 flex-wrap sm:flex-nowrap">
                    <div className="flex items-center gap-2 flex-shrink-0">
                        <Info className="h-4 w-4 flex-shrink-0 text-blue-600" />
                        <span className="font-semibold text-blue-900 text-sm">
                            Маршрут
                        </span>
                    </div>
                    <div
                        className="
                            grid grid-cols-1 gap-y-1.5 gap-x-5
                            min-[480px]:grid-cols-2
                            md:flex md:flex-1 md:flex-wrap md:items-center
                            md:gap-x-6 md:gap-y-1.5
                            lg:gap-x-8
                            min-w-0
                        "
                    >
                        <Metric
                            icon={<Navigation className="h-3.5 w-3.5 text-blue-600" />}
                            label="Расстояние"
                            value={`${info.totalDistance} км`}
                            hint="Суммарное расстояние по дорогам между всеми точками маршрута"
                        />
                        <Metric
                            icon={<Route className="h-3.5 w-3.5 text-blue-600" />}
                            label="Время в пути"
                            value={`${info.totalTravelTime} мин`}
                            hint="Суммарное время переездов между точками (без учёта выполнения задач)"
                        />
                        <Metric
                            icon={<AudioWaveform className="h-3.5 w-3.5 text-blue-600" />}
                            label="Общее время"
                            value={`${info.totalDuration} мин`}
                            hint="Время в пути + суммарная длительность выполнения всех задач"
                        />
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
