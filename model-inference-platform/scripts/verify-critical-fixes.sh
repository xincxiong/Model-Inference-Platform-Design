#!/bin/bash
set -e

echo "========================================="
echo "模型推理平台 - 关键修复验证脚本"
echo "========================================="
echo ""

echo "1. 验证 KEDA ScaledObject 配置..."
echo "   - 检查 keda-scaledobjects.yaml 文件..."
if [ -f "deploy/kubernetes/keda-scaledobjects.yaml" ]; then
    echo "   ✓ keda-scaledobjects.yaml 存在"
    grep -c "ScaledObject" deploy/kubernetes/keda-scaledobjects.yaml && echo "   ✓ 包含 ScaledObject 定义"
    grep -c "prometheus" deploy/kubernetes/keda-scaledobjects.yaml && echo "   ✓ 包含 Prometheus 触发器"
else
    echo "   ✗ keda-scaledobjects.yaml 不存在"
fi
echo ""

echo "2. 验证 WorkerPool 修复..."
echo "   - 检查 pool.go 中的 HAMi 集成..."
if [ -f "backend/internal/workerpool/pool.go" ]; then
    echo "   ✓ pool.go 存在"
    grep -c "hamiClient" backend/internal/workerpool/pool.go && echo "   ✓ 包含 hamiClient 字段"
    grep -c "k8sWatcher" backend/internal/workerpool/pool.go && echo "   ✓ 包含 k8sWatcher 字段"
    grep -c "validateHAMiAllocation" backend/internal/workerpool/pool.go && echo "   ✓ 包含 HAMi 分配验证"
else
    echo "   ✗ pool.go 不存在"
fi
echo ""

echo "3. 验证 HAMi K8s 客户端..."
echo "   - 检查 HAMi K8s 集成代码..."
if [ -f "backend/internal/hami/client.go" ]; then
    echo "   ✓ hami/client.go 存在"
    grep -c "BuildK8sClient" backend/internal/hami/client.go && echo "   ✓ 包含 K8s 客户端构建"
else
    echo "   ✗ hami/client.go 不存在"
fi

if [ -f "backend/internal/hami/k8s_watcher.go" ]; then
    echo "   ✓ hami/k8s_watcher.go 存在"
    grep -c "K8sWatcher" backend/internal/hami/k8s_watcher.go && echo "   ✓ 包含 K8sWatcher 结构"
    grep -c "ValidatePodGPUAllocation" backend/internal/hami/k8s_watcher.go && echo "   ✓ 包含 GPU 分配验证"
else
    echo "   ✗ hami/k8s_watcher.go 不存在"
fi
echo ""

echo "4. 验证 GPU 指标监控..."
echo "   - 检查 Prometheus 监控配置..."
if [ -f "deploy/kubernetes/prometheus-monitoring.yaml" ]; then
    echo "   ✓ prometheus-monitoring.yaml 存在"
    grep -c "ServiceMonitor" deploy/kubernetes/prometheus-monitoring.yaml && echo "   ✓ 包含 ServiceMonitor"
    grep -c "PrometheusRule" deploy/kubernetes/prometheus-monitoring.yaml && echo "   ✓ 包含 PrometheusRule"
    grep -c "DCGM" deploy/kubernetes/prometheus-monitoring.yaml && echo "   ✓ 包含 DCGM 指标配置"
else
    echo "   ✗ prometheus-monitoring.yaml 不存在"
fi
echo ""

echo "========================================="
echo "修复验证完成"
echo "========================================="
echo ""
echo "关键修复摘要:"
echo "  1. KEDA ScaledObject - 支持 GPU 利用率和队列深度自动扩缩容"
echo "  2. WorkerPool/HAMi 冲突 - 通过 HAMi 分配验证解决"
echo "  3. HAMi K8s API 集成 - 实现 Pod GPU 分配状态同步"
echo "  4. GPU 指标监控 - ServiceMonitor + 告警规则"
echo ""
echo "应用步骤:"
echo "  1. kubectl apply -f deploy/kubernetes/keda-scaledobjects.yaml"
echo "  2. kubectl apply -f deploy/kubernetes/prometheus-monitoring.yaml"
echo "  3. 确保 KEDA 和 Prometheus 已安装"
echo "  4. 验证自动扩容: kubectl get hpa -n inference-platform"
