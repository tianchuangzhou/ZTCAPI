import React, { useEffect, useState } from 'react';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';
import { BarChart3, CircleDollarSign, MessageSquareWarning, Users } from 'lucide-react';
import { API } from '../../helpers';

const OperationsReportPanel = ({ isAdminUser }) => {
  const [report, setReport] = useState(null);
  useEffect(() => {
    if (!isAdminUser) return undefined;
    let active = true;
    API.get('/api/operations/report?days=7', { skipErrorHandler: true })
      .then((response) => {
        if (active && response?.data?.success) setReport(response.data.data);
      })
      .catch(() => {});
    return () => { active = false; };
  }, [isAdminUser]);

  if (!isAdminUser || !report) return null;
  const usage = report.usage || {};
  const payments = report.payments || {};
  const metric = (Icon, label, value, color) => (
    <div className='flex items-center gap-3'>
      <div className={`flex items-center justify-center w-9 h-9 rounded-lg ${color}`}><Icon size={18} /></div>
      <div><Typography.Text type='tertiary' size='small'>{label}</Typography.Text><div className='text-lg font-semibold'>{value ?? 0}</div></div>
    </div>
  );
  return (
    <Card className='mb-4' title='运营报表 · 近 7 天' bordered>
      <div className='grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4'>
        {metric(BarChart3, '调用次数', usage.calls, 'bg-blue-50 text-blue-600')}
        {metric(Users, 'Token 用量', usage.tokens, 'bg-purple-50 text-purple-600')}
        {metric(CircleDollarSign, '消费额度', usage.quota, 'bg-green-50 text-green-600')}
        {metric(MessageSquareWarning, '错误率', `${((usage.error_rate || 0) * 100).toFixed(2)}%`, 'bg-orange-50 text-orange-600')}
      </div>
      <div className='grid grid-cols-1 lg:grid-cols-3 gap-4'>
        <div><div className='font-medium mb-2'>模型排行榜</div>{(report.model_ranking || []).slice(0, 5).map((row, i) => <div key={row.name || i} className='flex justify-between text-sm py-1'><span>{i + 1}. {row.name || '未知模型'}</span><span className='text-gray-500'>{row.calls} 次</span></div>)}</div>
        <div><div className='font-medium mb-2'>用户消费排行</div>{(report.user_ranking || []).slice(0, 5).map((row, i) => <div key={row.name || i} className='flex justify-between text-sm py-1'><span>{i + 1}. {row.name || '匿名用户'}</span><span className='text-gray-500'>{row.quota}</span></div>)}</div>
        <div><div className='font-medium mb-2'>支付与客服</div><div className='flex flex-wrap gap-2 mb-2'><Tag color='orange'>待支付 {payments.pending || 0}</Tag><Tag color='green'>已支付 {payments.successful || 0}</Tag><Tag color='red'>退款待处理 {payments.refund_requested || 0}</Tag></div><Typography.Text type='tertiary' size='small'>{report.operations?.support_url ? `客服入口：${report.operations.support_url}` : '未配置客服入口（SUPPORT_URL）'}</Typography.Text></div>
      </div>
    </Card>
  );
};

export default OperationsReportPanel;
