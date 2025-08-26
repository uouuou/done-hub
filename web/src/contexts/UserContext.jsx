// contexts/User/index.jsx
import React, { useEffect, useCallback, createContext, useState } from 'react';
import { useSelector } from 'react-redux';
import useLogin from 'hooks/useLogin';

export const UserContext = createContext();

// eslint-disable-next-line
const UserProvider = ({ children }) => {
  const [isUserLoaded, setIsUserLoaded] = useState(false);
  const account = useSelector((state) => state.account);
  const { loadUser: loadUserAction, loadUserGroup: loadUserGroupAction } = useLogin();

  const loadUser = useCallback(async () => {
    setIsUserLoaded(false);
    await loadUserAction();
    setIsUserLoaded(true);
  }, [loadUserAction]);

  const loadUserGroup = useCallback(() => {
    loadUserGroupAction();
  }, [loadUserGroupAction]);

  useEffect(() => {
    // 只有在没有用户信息时才加载
    if (!account.user) {
      // 静默加载用户信息，不显示错误
      loadUser().catch(() => {
        // 静默处理错误，避免在登录页面显示错误信息
      });
      loadUserGroup();
    } else {
      // 如果已经有用户信息，直接设置为已加载
      setIsUserLoaded(true);
    }
  }, [loadUser, loadUserGroup, account.user]);

  return <UserContext.Provider value={{ loadUser, isUserLoaded, loadUserGroup }}> {children} </UserContext.Provider>;
};

export default UserProvider;
