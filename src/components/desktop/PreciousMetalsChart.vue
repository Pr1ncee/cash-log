<template>
    <div>
        <div v-if="purchaseMarkersSummary.length" class="purchase-summary-container mb-2">
            <div v-for="(summary, idx) in purchaseMarkersSummary" :key="idx" class="purchase-summary-item d-inline-flex align-center me-6">
                <span class="purchase-dot"></span>
                <span class="text-body-2">{{ summary.description }} ({{ summary.date }})</span>
                <span class="text-body-2 ms-2 font-weight-bold" :class="summary.profitLoss >= 0 ? 'text-income' : 'text-expense'">
                    {{ summary.profitLoss >= 0 ? '+' : '' }}${{ summary.profitLoss.toFixed(2) }}/g
                    ({{ summary.profitLossPct >= 0 ? '+' : '' }}{{ summary.profitLossPct.toFixed(2) }}%)
                </span>
            </div>
        </div>
        <v-chart autoresize class="precious-metals-chart-container" :option="chartOptions" />
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useTheme } from 'vuetify';

import { ThemeType } from '@/core/theme.ts';

import type { PreciousMetalPriceDataPoint, PurchaseMarker } from '@/models/precious_metals.ts';

const props = defineProps<{
    historicalData: PreciousMetalPriceDataPoint[];
    currency: string;
    currentPrice: number;
    purchaseMarkers?: PurchaseMarker[];
}>();

const theme = useTheme();

const isDarkMode = computed<boolean>(() => theme.global.name.value === ThemeType.Dark);

interface PurchaseMarkerSummary {
    description: string;
    date: string;
    buyPrice: number;
    profitLoss: number;
    profitLossPct: number;
}

const purchaseMarkersSummary = computed<PurchaseMarkerSummary[]>(() => {
    if (!props.purchaseMarkers || !props.purchaseMarkers.length || !props.historicalData.length || !props.currentPrice) {
        return [];
    }

    const summaries: PurchaseMarkerSummary[] = [];

    for (const marker of props.purchaseMarkers) {
        // Find the gold price at the time of purchase (closest data point)
        let closestPrice = props.historicalData[0]!.price;
        let minDiff = Number.MAX_SAFE_INTEGER;

        for (const dp of props.historicalData) {
            const diff = Math.abs(dp.timestamp - marker.timestamp);
            if (diff < minDiff) {
                minDiff = diff;
                closestPrice = dp.price;
            }
        }

        const profitLoss = props.currentPrice - closestPrice;
        const profitLossPct = closestPrice !== 0 ? (profitLoss / closestPrice) * 100 : 0;

        const markerDate = new Date(marker.timestamp * 1000);
        summaries.push({
            description: marker.description,
            date: markerDate.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' }),
            buyPrice: closestPrice,
            profitLoss,
            profitLossPct
        });
    }

    return summaries;
});

const chartOptions = computed<object>(() => {
    const lineData: [number, number][] = [];

    for (const dp of props.historicalData) {
        lineData.push([dp.timestamp * 1000, dp.price]);
    }

    const seriesData: object[] = [
        {
            name: 'Gold Price',
            type: 'line',
            data: lineData,
            smooth: true,
            symbol: 'none',
            lineStyle: {
                width: 2,
                color: '#ffd700'
            },
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: isDarkMode.value ? 'rgba(255, 215, 0, 0.3)' : 'rgba(255, 215, 0, 0.4)' },
                        { offset: 1, color: isDarkMode.value ? 'rgba(255, 215, 0, 0.02)' : 'rgba(255, 215, 0, 0.05)' }
                    ]
                }
            },
            animation: true
        }
    ];

    // Build purchase marker scatter data
    const purchaseMarkerDetails: { description: string; amount: number; date: string; buyPrice: number }[] = [];

    if (props.purchaseMarkers && props.purchaseMarkers.length) {
        const scatterData: [number, number][] = [];

        for (const marker of props.purchaseMarkers) {
            // Find the closest price at purchase time
            let closestPrice = 0;
            let minDiff = Number.MAX_SAFE_INTEGER;

            for (const dp of props.historicalData) {
                const diff = Math.abs(dp.timestamp - marker.timestamp);
                if (diff < minDiff) {
                    minDiff = diff;
                    closestPrice = dp.price;
                }
            }

            const markerDate = new Date(marker.timestamp * 1000);
            purchaseMarkerDetails.push({
                description: marker.description,
                amount: marker.amount,
                date: markerDate.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' }),
                buyPrice: closestPrice
            });

            // Place at exact timestamp with the closest known price
            scatterData.push([marker.timestamp * 1000, closestPrice]);
        }

        seriesData.push({
            name: 'Purchases',
            type: 'scatter',
            data: scatterData,
            symbolSize: 14,
            itemStyle: {
                color: '#e53935',
                borderColor: '#fff',
                borderWidth: 2
            },
            z: 10
        });
    }

    return {
        tooltip: {
            trigger: 'axis',
            axisPointer: {
                type: 'cross',
                label: {
                    backgroundColor: isDarkMode.value ? '#333' : '#fff',
                    color: isDarkMode.value ? '#eee' : '#333'
                }
            },
            backgroundColor: isDarkMode.value ? '#333' : '#fff',
            borderColor: isDarkMode.value ? '#333' : '#fff',
            textStyle: {
                color: isDarkMode.value ? '#eee' : '#333'
            },
            formatter: (params: { seriesName: string; name: string; value: [number, number]; color: string; dataIndex: number }[]) => {
                if (!params || !params.length) return '';
                // Use the first param's timestamp for the header date
                const firstParam = params[0];
                const headerDate = firstParam ? new Date(firstParam.value[0]).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' }) : '';
                let tooltip = `${headerDate}<br/>`;

                for (const p of params) {
                    if (p.seriesName === 'Purchases') {
                        const detail = purchaseMarkerDetails[p.dataIndex];
                        if (detail) {
                            const pnl = props.currentPrice - detail.buyPrice;
                            const pnlPct = detail.buyPrice !== 0 ? (pnl / detail.buyPrice) * 100 : 0;
                            const pnlColor = pnl >= 0 ? '#4caf50' : '#e53935';
                            tooltip += `<span class="chart-pointer" style="background-color: ${p.color}"></span>`;
                            tooltip += `<span>${detail.description}</span>`;
                            tooltip += `<span class="ms-5" style="float: inline-end">$${detail.amount.toFixed(2)} on ${detail.date}</span><br/>`;
                            tooltip += `<span class="chart-pointer" style="background-color: ${pnlColor}"></span>`;
                            tooltip += `<span>P&L</span>`;
                            tooltip += `<span class="ms-5" style="float: inline-end; color: ${pnlColor}">${pnl >= 0 ? '+' : ''}$${pnl.toFixed(2)}/g (${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%)</span><br/>`;
                        }
                    } else {
                        const value = p.value[1] ?? 0;
                        tooltip += `<span class="chart-pointer" style="background-color: ${p.color}"></span>`;
                        tooltip += `<span>${p.seriesName}</span>`;
                        tooltip += `<span class="ms-5" style="float: inline-end">$${value.toFixed(2)}/g</span><br/>`;
                    }
                }

                return tooltip;
            }
        },
        legend: {
            orient: 'horizontal',
            top: 0,
            data: seriesData.map(s => (s as { name: string }).name),
            textStyle: {
                color: isDarkMode.value ? '#eee' : '#333'
            }
        },
        grid: {
            left: 80,
            right: 20,
            bottom: 40,
            top: 40
        },
        xAxis: [{
            type: 'time',
            axisLabel: {
                color: isDarkMode.value ? '#888' : '#666'
            }
        }],
        yAxis: [{
            type: 'value',
            axisLabel: {
                color: isDarkMode.value ? '#888' : '#666',
                formatter: (value: number) => `$${value.toFixed(0)}`
            },
            splitLine: {
                lineStyle: {
                    color: isDarkMode.value ? '#4f4f4f' : '#e1e6f2'
                }
            }
        }],
        series: seriesData,
        dataZoom: [{
            type: 'inside',
            start: 0,
            end: 100
        }]
    };
});
</script>

<style scoped>
.precious-metals-chart-container {
    width: 100%;
    height: 720px;
    margin-top: 10px;
}

@media (min-width: 600px) {
    .precious-metals-chart-container {
        height: 790px;
    }
}

.purchase-summary-container {
    padding: 8px 16px;
}

.purchase-dot {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: #e53935;
    margin-right: 6px;
}
</style>
