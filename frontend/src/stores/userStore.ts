import {reactive}from'vue';import type{User}from'../types';export const userStore=reactive<{current:User|null}>({current:null});export function setCurrentUser(v:User){userStore.current=v}
