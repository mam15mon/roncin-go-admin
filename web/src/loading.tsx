import { Flex } from 'antd';
import type React from 'react';
import { QuarterRing } from '@/components/ui';

const Loading: React.FC = () => (
  <Flex
    align="center"
    justify="center"
    style={{ minHeight: '60vh', width: '100%' }}
  >
    <QuarterRing size={32} strokeWidth="3px" className="text-primary" />
  </Flex>
);

export default Loading;
