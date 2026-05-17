export interface PreciousMetalPriceDataPoint {
    timestamp: number;
    price: number;
    date: string;
}

export interface PreciousMetalStatistics {
    high: number;
    low: number;
    open: number;
    close: number;
    average: number;
    change: number;
    changePct: number;
}

export interface PreciousMetalPriceResponse {
    metal: string;
    currency: string;
    unit: string;
    timeline: string;
    currentPrice: number;
    historicalData: PreciousMetalPriceDataPoint[];
    statistics: PreciousMetalStatistics | null;
    dataSource: string;
}

export interface PreciousMetalPriceRequest {
    metal: string;
    currency: string;
    unit: string;
    timeline: string;
}

export interface PreciousMetalRefreshRequest {
    metal: string;
    currency: string;
}

export interface PreciousMetalPortfolioItem {
    metal: string;
    currency: string;
    totalQuantity: number;
    unit: string;
    avgBuyPrice: number;
    currentPrice: number;
    totalCost: number;
    currentValue: number;
    profitLoss: number;
    profitLossPct: number;
}

export interface PreciousMetalPortfolioResponse {
    items: PreciousMetalPortfolioItem[];
    totalValue: number;
    totalCost: number;
}

export interface PurchaseMarker {
    timestamp: number;
    amount: number;
    description: string;
}
