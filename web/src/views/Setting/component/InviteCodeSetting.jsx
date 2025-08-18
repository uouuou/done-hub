import { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Grid,
  Checkbox,
  FormControlLabel,
  Tooltip,
  Alert
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material';
import { API } from 'utils/api';
import { showError, showSuccess, copy } from 'utils/common';
import { useTranslation } from 'react-i18next';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';

const InviteCodeSetting = () => {
  const { t } = useTranslation();
  const [inviteCodes, setInviteCodes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [openDialog, setOpenDialog] = useState(false);
  const [editingCode, setEditingCode] = useState(null);
  const [statistics, setStatistics] = useState({});
  const [inviteCodeRegisterEnabled, setInviteCodeRegisterEnabled] = useState(false);

  const [formData, setFormData] = useState({
    name: '',
    max_uses: 1,
    expires_at: null,
    count: 1,
    status: 1
  });

  useEffect(() => {
    loadInviteCodes();
    loadStatistics();
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, data } = res.data;
      if (success) {
        const setting = data.find(item => item.key === 'InviteCodeRegisterEnabled');
        setInviteCodeRegisterEnabled(setting?.value === 'true');
      }
    } catch (error) {
      console.error('Failed to load settings:', error);
    }
  };

  const loadInviteCodes = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/invite-code/');
      const { success, data } = res.data;
      if (success) {
        setInviteCodes(data.data || []);
      } else {
        showError('加载邀请码列表失败');
      }
    } catch (error) {
      showError('加载邀请码列表失败');
    }
    setLoading(false);
  };

  const loadStatistics = async () => {
    try {
      const res = await API.get('/api/invite-code/statistics');
      const { success, data } = res.data;
      if (success) {
        setStatistics(data);
      }
    } catch (error) {
      console.error('Failed to load statistics:', error);
    }
  };

  const handleOpenDialog = (code = null) => {
    if (code) {
      setEditingCode(code);
      setFormData({
        name: code.name,
        max_uses: code.max_uses,
        expires_at: code.expires_at ? dayjs(code.expires_at * 1000) : null,
        count: 1,
        status: code.status
      });
    } else {
      setEditingCode(null);
      setFormData({
        name: '',
        max_uses: 1,
        expires_at: null,
        count: 1,
        status: 1
      });
    }
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
    setEditingCode(null);
  };

  const handleSubmit = async () => {
    try {
      const submitData = {
        ...formData,
        expires_at: formData.expires_at ? Math.floor(formData.expires_at.valueOf() / 1000) : 0
      };

      if (editingCode) {
        // 编辑
        const res = await API.put(`/api/invite-code/${editingCode.id}`, submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess('邀请码更新成功');
          loadInviteCodes();
          loadStatistics();
          handleCloseDialog();
        } else {
          showError(message || '更新失败');
        }
      } else {
        // 新建
        const res = await API.post('/api/invite-code/', submitData);
        const { success, message, data } = res.data;
        if (success) {
          showSuccess(`成功创建 ${data.count} 个邀请码`);
          loadInviteCodes();
          loadStatistics();
          handleCloseDialog();
        } else {
          showError(message || '创建失败');
        }
      }
    } catch (error) {
      showError('操作失败');
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定要删除这个邀请码吗？')) {
      return;
    }

    try {
      const res = await API.delete(`/api/invite-code/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess('删除成功');
        loadInviteCodes();
        loadStatistics();
      } else {
        showError(message || '删除失败');
      }
    } catch (error) {
      showError('删除失败');
    }
  };

  const handleCopyCode = (code) => {
    copy(code, '邀请码');
  };

  const toggleInviteCodeRegister = async () => {
    try {
      const newValue = !inviteCodeRegisterEnabled;
      const res = await API.put('/api/option/', {
        key: 'InviteCodeRegisterEnabled',
        value: newValue.toString()
      });
      const { success, message } = res.data;
      if (success) {
        setInviteCodeRegisterEnabled(newValue);
        showSuccess(newValue ? '邀请码注册已开启' : '邀请码注册已关闭');
      } else {
        showError(message || '设置失败');
      }
    } catch (error) {
      showError('设置失败');
    }
  };

  const getStatusChip = (status) => {
    return status === 1 ? (
      <Chip label="启用" color="success" size="small" />
    ) : (
      <Chip label="禁用" color="error" size="small" />
    );
  };

  const formatDate = (timestamp) => {
    if (!timestamp) return '永不过期';
    return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
  };

  const isExpired = (timestamp) => {
    if (!timestamp) return false;
    return dayjs(timestamp * 1000).isBefore(dayjs());
  };

  return (
    <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="zh-cn">
      <Card>
        <CardContent>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="h6">邀请码管理</Typography>
            <Box>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={inviteCodeRegisterEnabled}
                    onChange={toggleInviteCodeRegister}
                  />
                }
                label="开启邀请码注册"
              />
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => handleOpenDialog()}
                sx={{ ml: 1 }}
              >
                创建邀请码
              </Button>
              <IconButton onClick={loadInviteCodes} sx={{ ml: 1 }}>
                <RefreshIcon />
              </IconButton>
            </Box>
          </Box>

          {inviteCodeRegisterEnabled && (
            <Alert severity="info" sx={{ mb: 2 }}>
              邀请码注册已开启，用户注册时必须提供有效的邀请码。
            </Alert>
          )}

          {/* 统计信息 */}
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={3}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h6">{statistics.total_count || 0}</Typography>
                <Typography variant="body2" color="textSecondary">总邀请码</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={3}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h6">{statistics.enabled_count || 0}</Typography>
                <Typography variant="body2" color="textSecondary">启用中</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={3}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h6">{statistics.disabled_count || 0}</Typography>
                <Typography variant="body2" color="textSecondary">已禁用</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={3}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h6">{statistics.total_uses || 0}</Typography>
                <Typography variant="body2" color="textSecondary">总使用次数</Typography>
              </Paper>
            </Grid>
          </Grid>

          {/* 邀请码列表 */}
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>邀请码</TableCell>
                  <TableCell>名称</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>使用情况</TableCell>
                  <TableCell>过期时间</TableCell>
                  <TableCell>创建时间</TableCell>
                  <TableCell>操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {inviteCodes.map((code) => (
                  <TableRow key={code.id}>
                    <TableCell>
                      <Box display="flex" alignItems="center">
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                          {code.code}
                        </Typography>
                        <Tooltip title="复制邀请码">
                          <IconButton size="small" onClick={() => handleCopyCode(code.code)}>
                            <CopyIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    </TableCell>
                    <TableCell>{code.name || '-'}</TableCell>
                    <TableCell>
                      {getStatusChip(code.status)}
                      {isExpired(code.expires_at) && (
                        <Chip label="已过期" color="warning" size="small" sx={{ ml: 1 }} />
                      )}
                    </TableCell>
                    <TableCell>
                      {code.used_count} / {code.max_uses}
                    </TableCell>
                    <TableCell>{formatDate(code.expires_at)}</TableCell>
                    <TableCell>{formatDate(code.created_time)}</TableCell>
                    <TableCell>
                      <IconButton size="small" onClick={() => handleOpenDialog(code)}>
                        <EditIcon />
                      </IconButton>
                      <IconButton size="small" onClick={() => handleDelete(code.id)}>
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* 创建/编辑对话框 */}
      <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>{editingCode ? '编辑邀请码' : '创建邀请码'}</DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="名称/备注"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            margin="normal"
          />
          <TextField
            fullWidth
            label="最大使用次数"
            type="number"
            value={formData.max_uses}
            onChange={(e) => setFormData({ ...formData, max_uses: parseInt(e.target.value) || 1 })}
            margin="normal"
            inputProps={{ min: 1 }}
          />
          {!editingCode && (
            <TextField
              fullWidth
              label="创建数量"
              type="number"
              value={formData.count}
              onChange={(e) => setFormData({ ...formData, count: parseInt(e.target.value) || 1 })}
              margin="normal"
              inputProps={{ min: 1, max: 100 }}
              helperText="一次最多创建100个邀请码"
            />
          )}
          <FormControl fullWidth margin="normal">
            <InputLabel>状态</InputLabel>
            <Select
              value={formData.status}
              onChange={(e) => setFormData({ ...formData, status: e.target.value })}
            >
              <MenuItem value={1}>启用</MenuItem>
              <MenuItem value={2}>禁用</MenuItem>
            </Select>
          </FormControl>
          <DateTimePicker
            label="过期时间"
            value={formData.expires_at}
            onChange={(newValue) => setFormData({ ...formData, expires_at: newValue })}
            slotProps={{
              textField: {
                fullWidth: true,
                margin: 'normal',
                helperText: '留空表示永不过期'
              }
            }}
            minDateTime={dayjs()}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>取消</Button>
          <Button onClick={handleSubmit} variant="contained">
            {editingCode ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>
    </LocalizationProvider>
  );
};

export default InviteCodeSetting;
