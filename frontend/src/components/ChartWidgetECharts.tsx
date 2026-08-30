"use client";

import React, { useRef, useEffect } from 'react';
import * as echarts from 'echarts';
import type { QueryResult, ChartType } from '@/lib/types';

interface ChartWidgetEChartsProps {
  result: QueryResult;
  chartType: ChartType;
  height?: number;
}

export default function ChartWidgetECharts({
  result,
  chartType,
  height = 260,
}: ChartWidgetEChartsProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!chartRef.current) return;
    chartInstance.current = echarts.init(chartRef.current);
    updateChart();
    const handleResize = () => chartInstance.current?.resize();
    window.addEventListener('resize', handleResize);
    return () => {
      chartInstance.current?.dispose();
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    updateChart();
  }, [result, chartType]);

  const updateChart = () => {
    if (!chartInstance.current || !result?.rows?.length) return;

    const data = result.rows.map((row) => {
      const obj: Record<string, any> = {};
      result.columns.forEach((col, i) => { obj[col] = row[i]; });
      return obj;
    });

    const numericCols = result.columns.filter(col =>
      data.some(d => typeof d[col] === 'number')
    );
    const categoryCols = result.columns.filter(col =>
      !numericCols.includes(col)
    );

    let xKey = categoryCols[0] || result.columns[0];
    let yKey = numericCols[0] || result.columns[1] || result.columns[0];
    let valueKey = numericCols[1] || numericCols[0];

    let option: echarts.EChartsOption = {};

    switch (chartType) {
      case 'line':
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: numericCols.map((k) => ({
            name: k,
            type: 'line',
            data: data.map(d => Number(d[k]) || 0),
            smooth: true,
          })),
        };
        break;

      case 'bar':
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: numericCols.map((k) => ({
            name: k,
            type: 'bar',
            data: data.map(d => Number(d[k]) || 0),
          })),
        };
        break;

      case 'pie':
        option = {
          tooltip: { trigger: 'item' },
          series: [{
            type: 'pie',
            radius: ['40%', '70%'],
            data: data.map(d => ({ name: String(d[xKey]), value: Number(d[yKey]) || 0 })),
            label: { show: true, formatter: '{b}: {d}%' },
          }],
        };
        break;

      case 'scatter':
        option = {
          tooltip: { trigger: 'item' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'value', name: xKey },
          yAxis: { type: 'value', name: yKey },
          series: [{
            type: 'scatter',
            data: data.map(d => [Number(d[xKey]), Number(d[yKey])]),
          }],
        };
        break;

      case 'heatmap': {
        const xCat = categoryCols[0] || result.columns[0];
        const yCat = categoryCols[1] || categoryCols[0] || result.columns[0];
        const valCol = numericCols[0] || result.columns[1];
        const xData = [...new Set(data.map(d => String(d[xCat])))];
        const yData = [...new Set(data.map(d => String(d[yCat])))];
        const heatData = data.map(d => [xData.indexOf(String(d[xCat])), yData.indexOf(String(d[yCat])), Number(d[valCol]) || 0]);
        option = {
          tooltip: { position: 'top' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: xData },
          yAxis: { type: 'category', data: yData },
          visualMap: {
            min: Math.min(...heatData.map(d => d[2])),
            max: Math.max(...heatData.map(d => d[2])),
            calculable: true,
            orient: 'horizontal',
            left: 'center',
            bottom: '0%',
          },
          series: [{ type: 'heatmap', data: heatData, label: { show: true } }],
        };
        break;
      }

      case 'treemap': {
        const nameCol = categoryCols[0] || result.columns[0];
        const valCol = numericCols[0] || result.columns[1];
        const treeData = data.map(d => ({ name: String(d[nameCol]), value: Number(d[valCol]) || 0 }));
        option = {
          tooltip: { trigger: 'item' },
          series: [{ type: 'treemap', data: treeData, leafDepth: 1, roam: false, nodeClick: false, label: { show: true, formatter: '{b}' } }],
        };
        break;
      }

      case 'boxplot': {
        const boxData = numericCols.map((key) => {
          const values = data.map(d => Number(d[key])).filter(v => !isNaN(v)).sort((a, b) => a - b);
          if (values.length === 0) return [0, 0, 0, 0, 0];
          const min = values[0];
          const max = values[values.length - 1];
          const median = values[Math.floor(values.length / 2)];
          const q1 = values[Math.floor(values.length / 4)];
          const q3 = values[Math.floor(3 * values.length / 4)];
          return [min, q1, median, q3, max];
        });
        option = {
          tooltip: { trigger: 'item' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: numericCols },
          yAxis: { type: 'value' },
          series: [{ type: 'boxplot', data: boxData }],
        };
        break;
      }

      case 'kpi': {
        const kpiValue = data.length > 0 ? Number(data[0][yKey]) : 0;
        option = {
          graphic: [{
            type: 'text',
            left: 'center',
            top: 'center',
            style: {
              text: kpiValue.toLocaleString(),
              fill: '#e6e9f0',
              font: 'bold 48px Arial, sans-serif',
              },
          }],
        };
        break;
      }

      case 'forecast': {
        const forecastCol = numericCols.find(c => c.toLowerCase().includes('forecast')) || numericCols[numericCols.length - 1];
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: numericCols.map((k) => ({
            name: k,
            type: 'line',
            data: data.map(d => Number(d[k]) || 0),
            smooth: true,
            lineStyle: { type: k === forecastCol ? 'dashed' : 'solid' },
          })),
        };
        break;
      }

      // --- НОВЫЕ ТИПЫ ---

      case 'combo': {
        // Комбинированный график: первый числовой столбец – bar, остальные – line
        const barCol = numericCols[0];
        const lineCols = numericCols.slice(1);
        const series: any[] = [
          {
            name: barCol,
            type: 'bar',
            data: data.map(d => Number(d[barCol]) || 0),
          },
          ...lineCols.map((k) => ({
            name: k,
            type: 'line',
            data: data.map(d => Number(d[k]) || 0),
            smooth: true,
          })),
        ];
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: series as any,
        };
        break;
      }

      case 'radar': {
        // Радар: первая категория – имя, остальные числовые – показатели
        const indicator = numericCols.map((k) => ({ name: k, max: Math.max(...data.map(d => Number(d[k]) || 0)) * 1.2 || 100 }));
        const radarData = data.map(d => ({
          name: String(d[xKey]),
          value: numericCols.map(k => Number(d[k]) || 0),
        }));
        option = {
          tooltip: { trigger: 'item' },
          radar: { indicator },
          series: [{
            type: 'radar',
            data: radarData,
          }],
        };
        break;
      }

      case 'funnel': {
        // Воронка: первый столбец – этап, второй – значение
        const funnelData = data.map(d => ({ name: String(d[xKey]), value: Number(d[yKey]) || 0 }));
        option = {
          tooltip: { trigger: 'item' },
          series: [{
            type: 'funnel',
            data: funnelData,
            sort: 'descending',
            label: { show: true, formatter: '{b}: {c}' },
          }],
        };
        break;
      }

      case 'waterfall': {
        // Водопад: первый столбец – категория, второй – значение
        const waterfallData = data.map(d => Number(d[yKey]) || 0);
        const total = waterfallData.reduce((a, b) => a + b, 0);
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: [{
            type: 'bar',
            data: waterfallData,
            stack: 'total',
          }],
        };
        break;
      }

      case 'bubble': {
        // Пузырьковая: x, y, size
        const sizeCol = numericCols[1] || numericCols[0];
        const bubbleData = data.map(d => ({
          value: [Number(d[xKey]), Number(d[yKey]), Number(d[sizeCol]) || 0],
        }));
        option = {
          tooltip: { trigger: 'item' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'value', name: xKey },
          yAxis: { type: 'value', name: yKey },
          series: [{
            type: 'scatter',
            data: bubbleData,
            symbolSize: (val: any) => Math.sqrt(val[2] || 0) * 5,
          }],
        };
        break;
      }

      case 'stackedbar': {
        // Столбчатая с накоплением
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: numericCols.map((k) => ({
            name: k,
            type: 'bar',
            stack: 'total',
            data: data.map(d => Number(d[k]) || 0),
          })),
        };
        break;
      }

      default:
        option = {
          tooltip: { trigger: 'axis' },
          grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
          xAxis: { type: 'category', data: data.map(d => String(d[xKey])) },
          yAxis: { type: 'value' },
          series: [{
            type: 'line',
            data: data.map(d => Number(d[yKey]) || 0),
          }],
        };
    }

    chartInstance.current.setOption(option, true);
    chartInstance.current.resize();
  };

  if (!result?.rows?.length) return <p className="text-sm text-slate-500">No data for this chart.</p>;

  return <div ref={chartRef} style={{ width: '100%', height: `${height}px` }} />;
}
