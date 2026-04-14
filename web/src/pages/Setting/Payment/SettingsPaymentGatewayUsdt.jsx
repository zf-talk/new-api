/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState, useRef } from 'react';
import {
  Banner,
  Button,
  Form,
  Row,
  Col,
  Typography,
  Spin,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsPaymentGatewayUsdt(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    UsdtEnabled: false,
    UsdtApiUrl: '',
    UsdtApiAuthToken: '',
    UsdtCurrency: 'cny',
    UsdtNetwork: 'tron',
    UsdtToken: 'usdt',
    UsdtQuotaPerUnit: 500000,
    UsdtMinTopUp: 10,
    UsdtOrderTimeout: 15,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        UsdtEnabled:
          props.options.UsdtEnabled === 'true' ||
          props.options.UsdtEnabled === true,
        UsdtApiUrl: props.options.UsdtApiUrl || '',
        UsdtApiAuthToken: props.options.UsdtApiAuthToken || '',
        UsdtCurrency: props.options.UsdtCurrency || 'cny',
        UsdtNetwork: props.options.UsdtNetwork || 'tron',
        UsdtToken: props.options.UsdtToken || 'usdt',
        UsdtQuotaPerUnit:
          parseFloat(props.options.UsdtQuotaPerUnit) || 500000,
        UsdtMinTopUp: parseFloat(props.options.UsdtMinTopUp) || 10,
        UsdtOrderTimeout:
          parseInt(props.options.UsdtOrderTimeout) || 15,
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitUsdtSetting = async () => {
    // Validate required fields when enabling
    if (inputs.UsdtEnabled) {
      if (!inputs.UsdtApiUrl || inputs.UsdtApiUrl.trim() === '') {
        showError(t('请先配置 BEpusdt API 地址和认证令牌'));
        return;
      }
      if (!inputs.UsdtApiAuthToken || inputs.UsdtApiAuthToken.trim() === '') {
        showError(t('请先配置 BEpusdt API 地址和认证令牌'));
        return;
      }
    }

    setLoading(true);
    try {
      const options = [];

      options.push({
        key: 'UsdtEnabled',
        value: inputs.UsdtEnabled ? 'true' : 'false',
      });

      if (inputs.UsdtApiUrl && inputs.UsdtApiUrl !== '') {
        options.push({ key: 'UsdtApiUrl', value: inputs.UsdtApiUrl });
      }

      if (inputs.UsdtApiAuthToken && inputs.UsdtApiAuthToken !== '') {
        options.push({
          key: 'UsdtApiAuthToken',
          value: inputs.UsdtApiAuthToken,
        });
      }

      options.push({
        key: 'UsdtCurrency',
        value: inputs.UsdtCurrency || 'cny',
      });
      options.push({
        key: 'UsdtNetwork',
        value: inputs.UsdtNetwork || 'tron',
      });
      options.push({
        key: 'UsdtToken',
        value: inputs.UsdtToken || 'usdt',
      });
      options.push({
        key: 'UsdtQuotaPerUnit',
        value: String(inputs.UsdtQuotaPerUnit || 500000),
      });
      options.push({
        key: 'UsdtMinTopUp',
        value: String(inputs.UsdtMinTopUp || 10),
      });
      options.push({
        key: 'UsdtOrderTimeout',
        value: String(inputs.UsdtOrderTimeout || 15),
      });

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({ ...inputs });
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('USDT 设置')}>
          <Text>
            {t(
              'USDT 充值通过 BEpusdt 支付网关实现，支持 TRC-20 等区块链网络的 USDT 收款。',
            )}
          </Text>
          <Banner
            type='info'
            description={t(
              '请先部署 BEpusdt 服务并获取 API 地址和认证令牌，启用前必须填写 API 地址和认证令牌。',
            )}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='UsdtEnabled'
                label={t('启用 USDT 充值')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UsdtApiUrl'
                label={t('BEpusdt API 地址')}
                placeholder={t('例如：http://your-server:8000')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='UsdtApiAuthToken'
                label={t('API 认证令牌')}
                placeholder={t('BEpusdt API Auth Token')}
                type='password'
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='UsdtCurrency'
                label={t('法币货币代码')}
                placeholder='cny'
                extraText={t('法币货币代码，默认 cny')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='UsdtNetwork'
                label={t('区块链网络')}
                placeholder='tron'
                extraText={t('区块链网络，默认 tron')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='UsdtToken'
                label={t('代币符号')}
                placeholder='usdt'
                extraText={t('代币符号，默认 usdt')}
              />
            </Col>
          </Row>

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='UsdtQuotaPerUnit'
                label={t('兑换比率')}
                placeholder='500000'
                min={0}
                step={1000}
                extraText={t('1 USDT 对应的额度数量，默认 500000')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='UsdtMinTopUp'
                label={t('最低充值金额')}
                placeholder='10'
                min={0}
                step={1}
                extraText={t('最低充值金额（法币），默认 10')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='UsdtOrderTimeout'
                label={t('订单超时时间（分钟）')}
                placeholder='15'
                min={1}
                step={1}
                extraText={t('订单超时时间，单位分钟，默认 15')}
              />
            </Col>
          </Row>

          <Button onClick={submitUsdtSetting} style={{ marginTop: 16 }}>
            {t('更新 USDT 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
