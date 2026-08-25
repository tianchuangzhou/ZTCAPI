import React from 'react';
import { Link } from 'react-router-dom';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';
import { ArrowRight, BookOpen, CheckCircle2, Copy, ShieldCheck } from 'lucide-react';
import { useContext } from 'react';
import { StatusContext } from '../../context/Status';
import { copy, showSuccess } from '../../helpers';

const Docs = () => {
  const [statusState] = useContext(StatusContext);
  const baseUrl = statusState?.status?.server_address || window.location.origin;
  const copyBaseUrl = async () => {
    if (await copy(baseUrl)) showSuccess('Base URL 已复制');
  };

  return (
    <main className='mt-[60px] px-4 pb-16 md:px-8'>
      <div className='mx-auto max-w-6xl'>
        <div className='mb-8 flex flex-wrap items-start justify-between gap-4'>
          <div>
            <div className='mb-3 flex items-center gap-2 text-semi-color-primary'>
              <BookOpen size={18} />
              <span className='text-sm font-semibold'>ZTC API 文档</span>
              <Tag size='small' color='green'>v1.1.0</Tag>
            </div>
            <Typography.Title heading={1} className='!mb-2'>统一接入，稳定调用</Typography.Title>
            <Typography.Text type='tertiary'>兼容 OpenAI、Claude 等常见协议，使用一个 Base URL 管理模型、密钥和用量。</Typography.Text>
          </div>
          <Link to='/pricing' className='flex items-center gap-1 text-sm text-semi-color-primary'>查看模型价格 <ArrowRight size={15} /></Link>
        </div>

        <div className='mb-6 grid gap-4 md:grid-cols-3'>
          <Card title='Base URL' headerExtraContent={<button type='button' aria-label='复制 Base URL' onClick={copyBaseUrl} className='text-semi-color-primary'><Copy size={16} /></button>}>
            <code className='break-all text-sm'>{baseUrl}</code>
          </Card>
          <Card title='鉴权方式'>
            <Typography.Text>在请求头中使用：</Typography.Text>
            <code className='mt-2 block break-all text-sm'>Authorization: Bearer sk-...</code>
          </Card>
          <Card title='封闭测试策略'>
            <div className='flex items-center gap-2 text-sm'><ShieldCheck size={16} className='text-green-600' /> 当前仅管理员创建测试用户</div>
          </Card>
        </div>

        <div className='grid gap-6 lg:grid-cols-2'>
          <Card title='OpenAI Chat Completions'>
            <pre className='overflow-x-auto rounded-md bg-gray-50 p-4 text-xs dark:bg-gray-900'>{`curl ${baseUrl}/v1/chat/completions \\
  -H "Authorization: Bearer $ZTC_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"你好"}]}'`}</pre>
          </Card>
          <Card title='Claude Messages'>
            <pre className='overflow-x-auto rounded-md bg-gray-50 p-4 text-xs dark:bg-gray-900'>{`curl ${baseUrl}/v1/messages \\
  -H "x-api-key: $ZTC_API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"claude-sonnet-4-5","max_tokens":256,"messages":[{"role":"user","content":"你好"}]}'`}</pre>
          </Card>
        </div>

        <Card className='mt-6' title='上线前约定'>
          <div className='grid gap-3 text-sm md:grid-cols-2'>
            {['每个模型先在渠道管理中完成真实请求测试', '按输入和输出 Token 配置价格后再开放调用', '429、超时和上游 5xx 会记录到消费与错误日志', '生产环境必须使用 HTTPS，不要在客户端暴露上游 Key'].map((item) => (
              <div key={item} className='flex items-start gap-2'><CheckCircle2 size={16} className='mt-0.5 shrink-0 text-green-600' /><span>{item}</span></div>
            ))}
          </div>
        </Card>
      </div>
    </main>
  );
};

export default Docs;
