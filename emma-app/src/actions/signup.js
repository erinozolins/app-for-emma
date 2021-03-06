// @flow
import {
  SIGNUP_FIRSTNAME_CHANGE,
  SIGNUP_LASTNAME_CHANGE,
  SIGNUP_PASSWORD_CHANGE,
  SIGNUP_EMAIL_CHANGE,
  SIGNUP_SEND_BEGIN,
  SIGNUP_SEND_SUCCESS,
  SIGNUP_SEND_FAIL
} from './../constants/action-types'
import { mainPage } from './navigation'
import { AsyncStorage } from 'react-native'
import { apiUrl } from './../../config'
import type { SignupAction } from './signup.js.flow'
import type { Dispatch } from './../types/Store'

export const updateFirstNameText = (payload: string): SignupAction => ({
  type: SIGNUP_FIRSTNAME_CHANGE,
  payload
})

export const updateLastNameText = (payload: string): SignupAction => ({
  type: SIGNUP_LASTNAME_CHANGE,
  payload
})

export const updatePasswordText = (payload: string): SignupAction => ({
  type: SIGNUP_PASSWORD_CHANGE,
  payload
})

export const updateEmailText = (payload: string): SignupAction => ({
  type: SIGNUP_EMAIL_CHANGE,
  payload
})

export const signupUser = (payload: Object): SignupAction => {
  return async (dispatch: Dispatch) => {
    dispatch({
      type: SIGNUP_SEND_BEGIN
    })
    let token
    try {
      const get = await fetch(`${apiUrl}/iphone-register`)
      const keys = await get.headers.forEach((val, key) => {
        if (key === 'csrf_token') {
          token = val
          AsyncStorage.setItem('@thisCoolApp:token', val)
        }
      })
    } catch (e) {
      dispatch(signupFail(e))
    }
    try {
      const call = await fetch(`${apiUrl}/iphone-register`, {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
          'X-CSRF-TOKEN': token
        },
        body: JSON.stringify({
          firstName: payload.firstName,
          lastName: payload.lastName,
          Email: payload.email,
          Password: payload.password
        })
      })
      const json = await call.json()
      dispatch(signupSuccess(json))
    } catch (e) {
      dispatch(signupFail(e))
    }
  }
}

const signupSuccess = payload => (dispatch: Dispatch) => {
  dispatch(mainPage())
  dispatch({
    type: SIGNUP_SEND_SUCCESS,
    payload
  })
}

export const signupFail = (e: Error) => (dispatch: Dispatch) => {
  console.log(e)
  dispatch({
    type: SIGNUP_SEND_FAIL
  })
}
