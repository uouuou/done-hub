import PropTypes from 'prop-types';
import * as Yup from 'yup';
import { Formik } from 'formik';
import { useTheme } from '@mui/material/styles';
import { useState, useEffect } from 'react';
import dayjs from 'dayjs';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Divider,
  Alert,
  FormControl,
  InputLabel,
  OutlinedInput,
  InputAdornment,
  Switch,
  FormControlLabel,
  FormHelperText,
  Select,
  MenuItem,
  Typography,
  Chip,
  Box,
  Autocomplete,
  TextField
} from '@mui/material';

import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { renderQuotaWithPrompt, showSuccess, showError } from 'utils/common';
import { API } from 'utils/api';
import { useTranslation } from 'react-i18next';
import 'dayjs/locale/zh-cn';

const validationSchema = Yup.object().shape({
  is_edit: Yup.boolean(),
  name: Yup.string().required('名称 不能为空'),
  remain_quota: Yup.number().min(0, '必须大于等于0'),
  expired_time: Yup.number(),
  unlimited_quota: Yup.boolean(),
  setting: Yup.object().shape({
    heartbeat: Yup.object().shape({
      enabled: Yup.boolean(),
      timeout_seconds: Yup.number().when('enabled', {
        is: true,
        then: () => Yup.number().min(30, '时间 必须大于等于30秒').max(90, '时间 必须小于等于90秒').required('时间 不能为空'),
        otherwise: () => Yup.number()
      })
    }),
    models: Yup.array().of(Yup.string()),
    subnet: Yup.string().test('is-valid-subnet', '无效的子网格式', function (value) {
      if (!value || value === '') return true; // 允许为空
      // 简单的IP地址或CIDR验证
      const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;
      if (!ipRegex.test(value)) return false;

      // 验证IP地址范围
      const ipPart = value.split('/')[0];
      const parts = ipPart.split('.');
      if (parts.length !== 4) return false;

      for (let part of parts) {
        const num = parseInt(part);
        if (isNaN(num) || num < 0 || num > 255) return false;
      }

      // 如果有子网掩码，验证其范围
      if (value.includes('/')) {
        const mask = parseInt(value.split('/')[1]);
        if (isNaN(mask) || mask < 0 || mask > 32) return false;
      }

      return true;
    })
  })
});

const originInputs = {
  is_edit: false,
  name: '',
  remain_quota: 0,
  expired_time: -1,
  unlimited_quota: false,
  group: '',

  setting: {
    heartbeat: {
      enabled: false,
      timeout_seconds: 30
    },
    models: [],
    subnet: ''
  }
};

const EditModal = ({ open, tokenId, onCancel, onOk, userGroupOptions }) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const [inputs, setInputs] = useState(originInputs);
  const [availableModels, setAvailableModels] = useState([]);
  const [loadingModels, setLoadingModels] = useState(false);

  const submit = async (values, { setErrors, setStatus, setSubmitting }) => {
    setSubmitting(true);

    values.remain_quota = parseInt(values.remain_quota);
    values.setting.heartbeat.timeout_seconds = parseInt(values.setting.heartbeat.timeout_seconds);
    let res;

    try {
      if (values.is_edit) {
        res = await API.put(`/api/token/`, { ...values, id: parseInt(tokenId) });
      } else {
        res = await API.post(`/api/token/`, values);
      }
      const { success, message } = res.data;
      if (success) {
        if (values.is_edit) {
          showSuccess('令牌更新成功！');
        } else {
          showSuccess('令牌创建成功，请在列表页面点击复制获取令牌！');
        }
        setSubmitting(false);
        setStatus({ success: true });
        onOk(true);
      } else {
        showError(message);
        setErrors({ submit: message });
      }
    } catch (error) {
      return;
    }
  };

  const loadToken = async () => {
    try {
      let res = await API.get(`/api/token/${tokenId}`);
      const { success, message, data } = res.data;
      if (success) {
        data.is_edit = true;
        setInputs(data);
      } else {
        showError(message);
      }
    } catch (error) {
      return;
    }
  };

  const loadAvailableModels = async () => {
    setLoadingModels(true);
    try {
      // 调用 /api/available_model 接口获取所有可用模型
      const res = await API.get('/api/available_model');
      const { success, message, data } = res.data;
      if (success && data) {
        const models = Object.keys(data);
        setAvailableModels(models);
      } else {
        showError(message || '获取可用模型失败');
      }
    } catch (error) {
      console.error('Failed to load models:', error);
      showError('获取可用模型失败');
    } finally {
      setLoadingModels(false);
    }
  };

  useEffect(() => {
    if (tokenId) {
      loadToken().then();
    } else {
      setInputs(originInputs);
    }
    loadAvailableModels();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tokenId]);

  return (
    <Dialog open={open} onClose={onCancel} fullWidth maxWidth={'md'}>
      <DialogTitle sx={{ margin: '0px', fontWeight: 700, lineHeight: '1.55556', padding: '24px', fontSize: '1.125rem' }}>
        {tokenId ? t('token_index.editToken') : t('token_index.createToken')}
      </DialogTitle>
      <Divider />
      <DialogContent>
        <Alert severity="info">{t('token_index.quotaNote')}</Alert>
        <Formik initialValues={inputs} enableReinitialize validationSchema={validationSchema} onSubmit={submit}>
          {({ errors, handleBlur, handleChange, handleSubmit, touched, values, setFieldError, setFieldValue, isSubmitting }) => (
            <form noValidate onSubmit={handleSubmit}>
              <FormControl fullWidth error={Boolean(touched.name && errors.name)} sx={{ ...theme.typography.otherInput }}>
                <InputLabel htmlFor="channel-name-label">{t('token_index.name')}</InputLabel>
                <OutlinedInput
                  id="channel-name-label"
                  label={t('token_index.name')}
                  type="text"
                  value={values.name}
                  name="name"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  inputProps={{ autoComplete: 'name' }}
                  aria-describedby="helper-text-channel-name-label"
                />
                {touched.name && errors.name && (
                  <FormHelperText error id="helper-tex-channel-name-label">
                    {errors.name}
                  </FormHelperText>
                )}
              </FormControl>
              {values.expired_time !== -1 && (
                <FormControl fullWidth error={Boolean(touched.expired_time && errors.expired_time)} sx={{ ...theme.typography.otherInput }}>
                  <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale={'zh-cn'}>
                    <DateTimePicker
                      label={t('token_index.expiryTime')}
                      ampm={false}
                      value={dayjs.unix(values.expired_time)}
                      onError={(newError) => {
                        if (newError === null) {
                          setFieldError('expired_time', null);
                        } else {
                          setFieldError('expired_time', t('token_index.invalidDate'));
                        }
                      }}
                      onChange={(newValue) => {
                        setFieldValue('expired_time', newValue.unix());
                      }}
                      slotProps={{
                        actionBar: {
                          actions: ['today', 'accept']
                        }
                      }}
                    />
                  </LocalizationProvider>
                  {errors.expired_time && (
                    <FormHelperText error id="helper-tex-channel-expired_time-label">
                      {errors.expired_time}
                    </FormHelperText>
                  )}
                </FormControl>
              )}
              <FormControlLabel
                control={
                  <Switch
                    checked={values.expired_time === -1}
                    onClick={() => {
                      if (values.expired_time === -1) {
                        setFieldValue('expired_time', Math.floor(Date.now() / 1000));
                      } else {
                        setFieldValue('expired_time', -1);
                      }
                    }}
                  />
                }
                label={t('token_index.neverExpires')}
              />

              <FormControl fullWidth error={Boolean(touched.remain_quota && errors.remain_quota)} sx={{ ...theme.typography.otherInput }}>
                <InputLabel htmlFor="channel-remain_quota-label">{t('token_index.quota')}</InputLabel>
                <OutlinedInput
                  id="channel-remain_quota-label"
                  label={t('token_index.quota')}
                  type="number"
                  value={values.remain_quota}
                  name="remain_quota"
                  endAdornment={<InputAdornment position="end">{renderQuotaWithPrompt(values.remain_quota)}</InputAdornment>}
                  onBlur={handleBlur}
                  onChange={handleChange}
                  aria-describedby="helper-text-channel-remain_quota-label"
                  disabled={values.unlimited_quota}
                />

                {touched.remain_quota && errors.remain_quota && (
                  <FormHelperText error id="helper-tex-channel-remain_quota-label">
                    {errors.remain_quota}
                  </FormHelperText>
                )}
              </FormControl>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={values.unlimited_quota === true}
                      onClick={() => {
                        setFieldValue('unlimited_quota', !values.unlimited_quota);
                      }}
                    />
                  }
                  label={t('token_index.unlimitedQuota')}
                />
              </FormControl>

              <Divider sx={{ margin: '16px 0px' }} />
              <Typography variant="h4">{t('token_index.heartbeat')}</Typography>
              <Typography variant="caption">{t('token_index.heartbeatTip')}</Typography>

              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={values?.setting?.heartbeat?.enabled === true}
                      onClick={() => {
                        setFieldValue('setting.heartbeat.enabled', !values.setting?.heartbeat?.enabled);
                      }}
                    />
                  }
                  label={t('token_index.heartbeat')}
                />
              </FormControl>

              {values?.setting?.heartbeat?.enabled && (
                <FormControl fullWidth>
                  <InputLabel>{t('token_index.heartbeatTimeout')}</InputLabel>
                  <OutlinedInput
                    id="channel-heartbeat-timeout-label"
                    label={t('token_index.heartbeatTimeout')}
                    type="number"
                    value={values?.setting?.heartbeat?.timeout_seconds}
                    onChange={(e) => {
                      setFieldValue('setting.heartbeat.timeout_seconds', e.target.value);
                    }}
                  />

                  {touched.setting?.heartbeat?.timeout_seconds && errors.setting?.heartbeat?.timeout_seconds ? (
                    <FormHelperText error id="helper-tex-channel-heartbeat-timeout-label">
                      {errors.setting?.heartbeat?.timeout_seconds}
                    </FormHelperText>
                  ) : (
                    <FormHelperText id="helper-tex-channel-heartbeat-timeout-label">
                      {t('token_index.heartbeatTimeoutHelperText')}
                    </FormHelperText>
                  )}
                </FormControl>
              )}

              <Divider sx={{ margin: '16px 0px' }} />

              <FormControl fullWidth>
                <InputLabel>{t('token_index.userGroup')}</InputLabel>
                <Select
                  label={t('token_index.userGroup')}
                  name="group"
                  value={values.group || '-1'}
                  onChange={(e) => {
                    const value = e.target.value === '-1' ? '' : e.target.value;
                    setFieldValue('group', value);
                  }}
                >
                  <MenuItem value="-1">跟随用户分组</MenuItem>
                  {userGroupOptions.map((option) => (
                    <MenuItem key={option.value} value={option.value}>
                      {option.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              <Divider sx={{ margin: '16px 0px' }} />
              <Typography
                variant="h4"
                sx={{
                  margin: '10px 0px'
                }}
              >
                {t('token_index.modelRestriction')}
              </Typography>

              <FormControl fullWidth>
                <Autocomplete
                  multiple
                  options={availableModels}
                  value={values.setting?.models || []}
                  onChange={(event, newValue) => {
                    setFieldValue('setting.models', newValue);
                  }}
                  loading={loadingModels}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      variant="outlined"
                      label={t('token_index.selectModels')}
                      placeholder={t('token_index.selectModelsPlaceholder')}
                      helperText={t('token_index.modelRestrictionTip')}
                    />
                  )}
                />
              </FormControl>

              <Divider sx={{ margin: '16px 0px' }} />
              <Typography variant="h4" sx={{ margin: '10px 0px' }}>
                {t('token_index.ipRestriction')}
              </Typography>

              <FormControl fullWidth error={Boolean(touched.setting?.subnet && errors.setting?.subnet)}>
                <InputLabel htmlFor="subnet-label">{t('token_index.ipRestriction')}</InputLabel>
                <OutlinedInput
                  id="subnet-label"
                  label={t('token_index.ipRestriction')}
                  type="text"
                  value={values.setting?.subnet || ''}
                  onChange={(e) => {
                    setFieldValue('setting.subnet', e.target.value);
                  }}
                  placeholder={t('token_index.subnetPlaceholder')}
                />
                {touched.setting?.subnet && errors.setting?.subnet && <FormHelperText error>{errors.setting?.subnet}</FormHelperText>}
                <FormHelperText>{t('token_index.subnetHelperText')}</FormHelperText>
              </FormControl>
              <DialogActions>
                <Button onClick={onCancel}>{t('token_index.cancel')}</Button>
                <Button disableElevation disabled={isSubmitting} type="submit" variant="contained" color="primary">
                  {t('token_index.submit')}
                </Button>
              </DialogActions>
            </form>
          )}
        </Formik>
      </DialogContent>
    </Dialog>
  );
};

export default EditModal;

EditModal.propTypes = {
  open: PropTypes.bool,
  tokenId: PropTypes.number,
  onCancel: PropTypes.func,
  onOk: PropTypes.func,
  userGroupOptions: PropTypes.array
};
