import React, { useEffect, useMemo, useState } from 'react';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';
import { AlertTriangle, CheckCircle2, ShieldCheck, XCircle } from 'lucide-react';
import { API } from '../../helpers';

const CATEGORY_LABELS = {
  security: '安全',
  billing: '计费',
  stability: '稳定性',
  operations: '运营',
};

const STATUS_META = {
  pass: { label: '通过', color: 'green', icon: CheckCircle2 },
  warn: { label: '注意', color: 'orange', icon: AlertTriangle },
  fail: { label: '阻断', color: 'red', icon: XCircle },
};

const ProductionReadinessPanel = ({ isAdminUser, t }) => {
  const [data, setData] = useState(null);

  useEffect(() => {
    if (!isAdminUser) return;
    let active = true;
    API.get('/api/production/readiness', { skipErrorHandler: true })
      .then((response) => {
        if (active && response?.data?.success) setData(response.data.data);
      })
      .catch(() => {
        // Non-root admins cannot access this root-only diagnostic; keep the dashboard clean.
      });
    return () => {
      active = false;
    };
  }, [isAdminUser]);

  const groupedChecks = useMemo(() => {
    if (!data?.checks) return [];
    return Object.entries(
      data.checks.reduce((groups, check) => {
        groups[check.category] = groups[check.category] || [];
        groups[check.category].push(check);
        return groups;
      }, {}),
    );
  }, [data]);

  if (!isAdminUser || !data) return null;

  const statusMeta = STATUS_META[data.status] || STATUS_META.warn;
  const StatusIcon = statusMeta.icon;

  return (
    <Card className='mb-4' title={null} bordered>
      <div className='flex flex-wrap items-center justify-between gap-3 mb-4'>
        <div className='flex items-center gap-3'>
          <div className='flex items-center justify-center w-9 h-9 rounded-lg bg-blue-50 dark:bg-blue-950/40'>
            <ShieldCheck size={20} className='text-blue-600' />
          </div>
          <div>
            <Typography.Title heading={5} className='!mb-0'>生产就绪检查</Typography.Title>
            <Typography.Text type='tertiary'>安全、计费、稳定性和运营的上线前检查</Typography.Text>
          </div>
        </div>
        <div className='flex items-center gap-2'>
          <Tag color={statusMeta.color} prefix={<StatusIcon size={13} />}>
            {statusMeta.label}
          </Tag>
          <Typography.Text type='tertiary' className='text-xs'>
            {data.summary.pass} 通过 · {data.summary.warn} 注意 · {data.summary.fail} 阻断
          </Typography.Text>
        </div>
      </div>

      <div className='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3'>
        {groupedChecks.map(([category, checks]) => {
          const worst = checks.some((check) => check.status === 'fail')
            ? 'fail'
            : checks.some((check) => check.status === 'warn')
              ? 'warn'
              : 'pass';
          const groupMeta = STATUS_META[worst];
          return (
            <div key={category} className='rounded-lg border border-solid border-gray-200 dark:border-gray-700 p-3'>
              <div className='flex items-center justify-between mb-2'>
                <strong>{CATEGORY_LABELS[category] || category}</strong>
                <Tag size='small' color={groupMeta.color}>{groupMeta.label}</Tag>
              </div>
              <div className='space-y-2'>
                {checks.map((check) => {
                  const meta = STATUS_META[check.status] || STATUS_META.warn;
                  const Icon = meta.icon;
                  return (
                    <div key={check.key} className='flex items-start gap-2 text-sm'>
                      <Icon size={15} className={`mt-0.5 ${check.status === 'pass' ? 'text-green-600' : check.status === 'fail' ? 'text-red-600' : 'text-orange-600'}`} />
                      <div className='min-w-0'>
                        <div className='font-medium'>{check.label}</div>
                        <div className='text-xs text-gray-500 dark:text-gray-400'>{check.detail}</div>
                        {check.action && <div className='text-xs text-blue-600 mt-1'>{check.action}</div>}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
};

export default ProductionReadinessPanel;
